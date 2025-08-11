package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/recera/vango/cmd/vango/internal/config"
	"github.com/recera/vango/cmd/vango/internal/pragma"
	"github.com/recera/vango/cmd/vango/internal/router"
	"github.com/recera/vango/cmd/vango/internal/routes"
	"github.com/recera/vango/cmd/vango/internal/tailwind"
	"github.com/recera/vango/cmd/vango/internal/template"
	"github.com/recera/vango/internal/assets"
	"github.com/recera/vango/internal/cache"
	"github.com/recera/vango/pkg/live"
	"github.com/spf13/cobra"
)

type devServer struct {
	port                 int
	host                 string
	watcher              *fsnotify.Watcher
	wsClients            map[*websocket.Conn]bool
	wsMutex              sync.RWMutex
	upgrader             websocket.Upgrader
	buildMutex           sync.Mutex
	lastBuild            time.Time
	buildCache           *cache.Cache
	tailwindRunner       *tailwind.Runner
	config               *config.Config
	disableTailwind      bool
	liveServer           *live.Server // Add live server for server-driven components
	routeHandler         http.Handler // Composite handler for routes
	routeCompiler        *routes.Compiler
	apiPatterns          []string
	serverDrivenPatterns []string
	ssrPagePatterns      []string
}

func newDevCommand() *cobra.Command {
	var port int
	var host string
	var cwd string
	var noTailwind bool

	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Start the development server",
		Long:  `Starts a development server with file watching, hot reloading, and live updates.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cwd != "" {
				if err := os.Chdir(cwd); err != nil {
					return fmt.Errorf("failed to change directory to %s: %w", cwd, err)
				}
			}
			if noTailwind {
				// set via env var consumed later
				os.Setenv("VANGO_NO_TAILWIND", "1")
			}
			return runDev(host, port)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 5173, "Port to run the dev server on")
	cmd.Flags().StringVarP(&host, "host", "H", "localhost", "Host to bind the dev server to")
	cmd.Flags().StringVar(&cwd, "cwd", "", "Working directory of the app (defaults to current)")
	cmd.Flags().BoolVar(&noTailwind, "no-tailwind", false, "Disable Tailwind CSS watcher")

	return cmd
}

func runDev(host string, port int) error {
	// Load configuration
	cfg, err := config.Load(".")
	if err != nil {
		log.Printf("⚠️  Failed to load vango.json: %v (using defaults)\n", err)
		cfg = config.DefaultConfig()
	}

	// Override port if provided via CLI (CLI takes precedence)
	if port != 0 {
		cfg.Dev.Port = port
	} else if cfg.Dev.Port != 0 {
		port = cfg.Dev.Port
	}

	// Override host if provided via CLI (CLI takes precedence)
	if host != "" {
		cfg.Dev.Host = host
	} else if cfg.Dev.Host != "" {
		host = cfg.Dev.Host
	}

	// Initialize build cache
	buildCache, err := cache.New(cache.DefaultConfig())
	if err != nil {
		log.Printf("⚠️  Failed to initialize build cache: %v", err)
		// Continue without cache
	}

	// Initialize live server for server-driven components
	log.Println("🔌 Initializing live protocol server...")
	liveServer := live.NewServer()
	live.InitBridge(liveServer)
	log.Println("✅ Live protocol server initialized")

	server := &devServer{
		port:       port,
		host:       host,
		wsClients:  make(map[*websocket.Conn]bool),
		buildCache: buildCache,
		config:     cfg,
		liveServer: liveServer,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in dev mode
				return true
			},
		},
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()
	server.watcher = watcher

	// Watch for Go and CSS files
	if err := server.setupWatcher(); err != nil {
		return fmt.Errorf("failed to setup watcher: %w", err)
	}

	// Start Tailwind runner if configured
	if !server.disableTailwind {
		if err := server.startTailwind(); err != nil {
			log.Printf("⚠️  Failed to start Tailwind: %v\n", err)
			// Continue without Tailwind
		}
	}

	// Scan and generate routes
	log.Println("🔍 Scanning routes...")
	routeManifest, err := router.ScanRoutes("app/routes")
	if err != nil {
		log.Printf("⚠️  Failed to scan routes: %v\n", err)
		// Continue without routes for now
	} else {
		log.Printf("  Found %d routes\n", routeManifest.TotalRoutes)

		// Generate route tree and helpers
		if err := router.GenerateRouteTree(routeManifest); err != nil {
			log.Printf("⚠️  Failed to generate routes: %v\n", err)
		}

		// Generate client route table
		if err := router.GenerateClientRouteTable(routeManifest); err != nil {
			log.Printf("⚠️  Failed to generate client route table: %v\n", err)
		}
	}

	// Initial build
	log.Println("🚀 Starting Vango dev server...")

	// Compile any existing VEX templates first
	log.Println("🎨 Compiling VEX templates...")
	if err := server.compileAllTemplates(); err != nil {
		log.Printf("⚠️  Template compilation warning: %v\n", err)
		// Continue even if template compilation fails
	}

	// Compile server routes
	log.Println("📍 Compiling server routes...")
	if err := server.compileRoutes(); err != nil {
		log.Printf("⚠️  Route compilation failed: %v\n", err)
		// Continue without routes for backwards compatibility
	}

	if err := server.buildWASM(); err != nil {
		return fmt.Errorf("initial build failed: %w", err)
	}

	// Start file watcher
	go server.watchFiles()

	// Set up HTTP routes
	mux := http.NewServeMux()

	// WebSocket endpoint for live updates
	mux.HandleFunc("/vango/live/", server.handleWebSocket)

	// Serve WASM files
	mux.HandleFunc("/app.wasm", server.serveWASM)
	mux.HandleFunc("/wasm_exec.js", server.serveWasmExec)

	// Serve bootstrap.js
	mux.HandleFunc("/vango/bootstrap.js", server.serveBootstrap)

	// Serve styles
	mux.HandleFunc("/styles.css", server.serveRootStyles)
	mux.HandleFunc("/styles/", server.serveStyles)

	// Serve router table for client-side routing
	mux.HandleFunc("/router/table.json", server.serveRouterTable)

	// Dynamic routing - use route handler if available, fallback to static
	if server.routeHandler != nil {
		// For client-side routes, we need to serve index.html so the WASM app can handle routing
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Check if this is a route that should be handled client-side
			path := r.URL.Path
			
			// Serve index.html for client-side routes (/, /about, /counter)
			// These will be handled by the WASM app
			if path == "/" || path == "/about" || path == "/counter" {
				// Serve the index.html file so WASM app can handle routing
				filePath := filepath.Join("public", "index.html")
				content, err := os.ReadFile(filePath)
				if err != nil {
					http.Error(w, "File not found", http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "text/html")
				w.Header().Set("Cache-Control", "no-cache")
				w.Write(content)
				return
			}
			
			// Let the route handler handle other routes (like server-driven ones)
			server.routeHandler.ServeHTTP(w, r)
		})
	} else {
		// Fallback to static serving for backwards compatibility
		mux.HandleFunc("/", server.serveStatic)
	}
	// Quiet favicon 404s during dev
	mux.HandleFunc("/favicon.ico", server.serveFavicon)

	// Start server
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("✨ Dev server running at http://%s\n", addr)

	// Set up graceful shutdown
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n🛑 Shutting down dev server...")

		// Stop Tailwind runner if it's running
		if server.tailwindRunner != nil && server.tailwindRunner.IsRunning() {
			server.tailwindRunner.Stop()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	return srv.ListenAndServe()
}

func (s *devServer) startTailwind() error {
	if os.Getenv("VANGO_NO_TAILWIND") == "1" {
		log.Println("📝 Tailwind disabled via --no-tailwind")
		return nil
	}
	// Check if Tailwind is enabled in config
	if s.config != nil && s.config.Styling != nil && s.config.Styling.Tailwind != nil {
		tw := s.config.Styling.Tailwind
		if tw.Enabled {
			// Create Tailwind runner with config
			s.tailwindRunner = tailwind.NewRunner(tailwind.Config{
				ConfigPath: tw.ConfigPath,
				InputPath:  tw.InputPath,
				OutputPath: tw.OutputPath,
				Watch:      tw.Watch,
				Strategy:   tw.Strategy,
				Version:    tw.Version,
				AutoDownload: func() bool {
					if tw.AutoDownload {
						return true
					}
					return true
				}(),
			})
			// Ensure initial build once so /styles.css exists
			if err := s.tailwindRunner.BuildOnce(); err != nil {
				log.Printf("⚠️  Tailwind initial build failed: %v\n", err)
			}
			return s.tailwindRunner.Start()
		}
	}

	// Fallback: Check if tailwind.config.js exists even if not configured
	if _, err := os.Stat("tailwind.config.js"); err == nil {
		log.Println("📝 Found tailwind.config.js, enabling Tailwind CSS...")
		s.tailwindRunner = tailwind.NewRunner(tailwind.Config{
			Watch:        true, // Enable watch mode by default in dev
			Strategy:     "auto",
			AutoDownload: true,
		})
		if err := s.tailwindRunner.BuildOnce(); err != nil {
			log.Printf("⚠️  Tailwind initial build failed: %v\n", err)
		}
		return s.tailwindRunner.Start()
	}

	return nil
}

func (s *devServer) setupWatcher() error {
	// Watch current directory and subdirectories
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and node_modules
		if info.IsDir() && (strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules") {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return s.watcher.Add(path)
		}

		return nil
	})

	return err
}

func (s *devServer) watchFiles() {
	debounce := time.NewTimer(0)
	<-debounce.C // drain initial timer

	var pendingEvents []fsnotify.Event
	var mu sync.Mutex

	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// Filter relevant files
			if !s.isRelevantFile(event.Name) {
				continue
			}

			mu.Lock()
			pendingEvents = append(pendingEvents, event)
			mu.Unlock()

			// Reset debounce timer
			debounce.Reset(100 * time.Millisecond)

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Println("Watcher error:", err)

		case <-debounce.C:
			mu.Lock()
			events := pendingEvents
			pendingEvents = nil
			mu.Unlock()

			if len(events) > 0 {
				s.handleFileChanges(events)
			}
		}
	}
}

func (s *devServer) isRelevantFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".go" || ext == ".css" || ext == ".js" || ext == ".html" || ext == ".vex"
}

func (s *devServer) handleFileChanges(events []fsnotify.Event) {
	var hasGoChanges, hasCSSChanges, hasVexChanges, hasRouteChanges bool
	var changedGoFiles []string
	var changedVexFiles []string

	for _, event := range events {
		ext := strings.ToLower(filepath.Ext(event.Name))
		switch ext {
		case ".go":
			hasGoChanges = true
			changedGoFiles = append(changedGoFiles, event.Name)
			// Check if this is a route file
			if strings.Contains(event.Name, "app/routes") {
				hasRouteChanges = true
			}
		case ".css":
			hasCSSChanges = true
		case ".vex":
			hasVexChanges = true
			changedVexFiles = append(changedVexFiles, event.Name)
			// VEX files in routes are also route changes
			if strings.Contains(event.Name, "app/routes") {
				hasRouteChanges = true
			}
		}
	}

	// Compile VEX templates first (they generate .go files)
	if hasVexChanges {
		log.Println("🎨 VEX templates changed, compiling...")
		for _, vexFile := range changedVexFiles {
			if err := s.compileTemplate(vexFile); err != nil {
				log.Printf("❌ Failed to compile template %s: %v\n", filepath.Base(vexFile), err)
				s.notifyClients("error", map[string]interface{}{
					"message": fmt.Sprintf("Template compilation failed: %v", err),
				})
			} else {
				log.Printf("✅ Compiled %s\n", filepath.Base(vexFile))
				// Mark that we have Go changes since we generated a .go file
				hasGoChanges = true
			}
		}
	}

	// Regenerate routes if route files changed
	if hasRouteChanges {
		log.Println("🔄 Routes changed, regenerating...")
		routeManifest, err := router.ScanRoutes("app/routes")
		if err != nil {
			log.Printf("❌ Failed to scan routes: %v\n", err)
		} else {
			// Generate route tree and helpers
			if err := router.GenerateRouteTree(routeManifest); err != nil {
				log.Printf("❌ Failed to generate routes: %v\n", err)
			} else {
				// Generate client route table
				if err := router.GenerateClientRouteTable(routeManifest); err != nil {
					log.Printf("❌ Failed to generate client route table: %v\n", err)
				} else {
					log.Printf("✅ Regenerated routes (%d routes)\n", routeManifest.TotalRoutes)
				}
			}
		}

		// Recompile server route handlers
		if err := s.recompileRoutes(); err != nil {
			log.Printf("❌ Failed to recompile routes: %v\n", err)
			s.notifyClients("error", map[string]interface{}{
				"message": fmt.Sprintf("Route compilation failed: %v", err),
			})
		} else {
			// Notify clients to reload for new routes
			s.notifyClients("reload", map[string]interface{}{
				"target": "routes",
				"reason": "Route handlers updated",
			})
		}
	}

	if hasGoChanges {
		// Invalidate cache for changed files
		if s.buildCache != nil {
			for _, file := range changedGoFiles {
				count := s.buildCache.InvalidateByDependency(file)
				if count > 0 {
					log.Printf("🗑️  Invalidated %d cached builds due to %s", count, filepath.Base(file))
				}
			}
		}

		log.Println("🔄 Go files changed, rebuilding WASM...")
		if err := s.buildWASM(); err != nil {
			log.Printf("❌ Build failed: %v\n", err)
			s.notifyClients("error", map[string]interface{}{
				"message": fmt.Sprintf("Build failed: %v", err),
			})
		} else {
			log.Println("✅ Build succeeded, reloading...")
			s.notifyClients("reload", map[string]interface{}{
				"target": "wasm",
			})
		}
	}

	if hasCSSChanges {
		log.Println("🎨 CSS files changed, reloading styles...")
		s.notifyClients("reload", map[string]interface{}{
			"target": "css",
		})
	}
}

func (s *devServer) buildWASM() error {
	s.buildMutex.Lock()
	defer s.buildMutex.Unlock()

	// Run pragma scanner to detect and inject build tags
	scanner, err := pragma.NewScanner(pragma.ScannerConfig{
		AutoInjectTags: true,
		Verbose:        false,
		CacheDir:       filepath.Join(os.Getenv("HOME"), ".cache", "vango", "pragma"),
	})
	if err != nil {
		log.Printf("⚠️  Pragma scanner failed: %v\n", err)
		// Continue without pragma scanning
	} else {
		manifest, err := scanner.Scan()
		if err != nil {
			log.Printf("⚠️  Pragma scan failed: %v\n", err)
		} else {
			// Log pragma scan results in dev mode
			if len(manifest.ServerFiles) > 0 || len(manifest.ClientFiles) > 0 {
				log.Printf("📝 Pragma scan: %d server, %d client, %d shared files",
					len(manifest.ServerFiles), len(manifest.ClientFiles), len(manifest.SharedFiles))
			}
		}
	}

	// Build WASM with TinyGo (client-side code)
	log.Println("🔨 Building WASM with TinyGo...")

	// Ensure output directory exists
	os.MkdirAll("public", 0755)

	// Try to use cached build if available
	wasmPath := "public/app.wasm"
	cacheUsed := false

	if s.buildCache != nil {
		// Generate cache key from source files
		cacheKey, err := generateWASMCacheKey()
		if err == nil {
			// Check cache
			if cachedWASM, found := s.buildCache.Get(cacheKey); found {
				// Use cached WASM
				if err := os.WriteFile(wasmPath, cachedWASM, 0644); err == nil {
					log.Println("⚡ Using cached WASM build")
					cacheUsed = true
				}
			}

			if !cacheUsed {
				// Build and cache
				// Try to find the WASM main file
				wasmMainPath := "./app/client/main.go"
				if _, err := os.Stat(wasmMainPath); os.IsNotExist(err) {
					// Fallback to app/main.go if client/main.go doesn't exist
					wasmMainPath = "./app/main.go"
				}

				cmd := exec.Command("tinygo", "build",
					"-o", wasmPath,
					"-target", "wasm",
					"-tags", "vango_client", // Build with client tag for WASM
					"-no-debug",
					"-opt", "2",
					wasmMainPath,
				)

				output, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("TinyGo build failed: %w\nOutput: %s", err, output)
				}

				// Cache the built WASM
				if wasmData, err := os.ReadFile(wasmPath); err == nil {
					deps := collectGoFiles("./app")
					s.buildCache.PutWithDeps(cacheKey, wasmData, deps)
					log.Println("💾 Cached WASM build")
				}
			}
		} else {
			// Fallback to normal build without cache
			log.Printf("⚠️  Cache key generation failed: %v", err)
			cacheUsed = false
		}
	}

	// If cache wasn't used, do normal build
	if !cacheUsed && s.buildCache == nil {
		// Try to find the WASM main file
		wasmMainPath := "./app/client/main.go"
		if _, err := os.Stat(wasmMainPath); os.IsNotExist(err) {
			// Fallback to app/main.go if client/main.go doesn't exist
			wasmMainPath = "./app/main.go"
		}

		cmd := exec.Command("tinygo", "build",
			"-o", wasmPath,
			"-target", "wasm",
			"-tags", "vango_client",
			"-no-debug",
			"-opt", "2",
			wasmMainPath,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("TinyGo build failed: %w\nOutput: %s", err, output)
		}
	}

	// Update last build time
	s.lastBuild = time.Now()

	// Get WASM size
	if info, err := os.Stat("public/app.wasm"); err == nil {
		size := info.Size()
		log.Printf("📦 WASM size: %.2f KB\n", float64(size)/1024)
	}

	return nil
}

// compileTemplate compiles a single VEX template file
func (s *devServer) compileTemplate(vexFile string) error {
	return template.ProcessTemplateFile(vexFile)
}

// compileAllTemplates compiles all VEX templates in the project
func (s *devServer) compileAllTemplates() error {
	// Check common locations for templates
	locations := []string{"app", "app/routes", "app/components"}

	for _, dir := range locations {
		if _, err := os.Stat(dir); err == nil {
			if err := template.ProcessDirectory(dir); err != nil {
				return fmt.Errorf("failed to process templates in %s: %w", dir, err)
			}
		}
	}

	return nil
}

func (s *devServer) compileRoutes() error {
	// Create a fallback handler that serves static files
	staticHandler := http.HandlerFunc(s.serveStatic)

	// Refresh route classifiers
	_ = s.refreshRouteClassifiers()

	// Preferred: compiler for API + SSR/universal routes
	var compiledHandler http.Handler
	if comp, err := routes.NewCompiler("app/routes"); err == nil {
		if h, err := comp.CompileAll(); err == nil && h != nil {
			s.routeCompiler = comp
			compiledHandler = h
		} else if err != nil {
			log.Printf("⚠️  Compiler failed, falling back: %v", err)
		}
	} else {
		log.Printf("⚠️  NewCompiler failed, falling back: %v", err)
	}

	// Live handler for server-driven routes
	var liveOrStaticHandler http.Handler = staticHandler
	if s.liveServer != nil {
		if liveHandler, err := routes.NewLiveHandler("app/routes", s.liveServer, staticHandler); err == nil {
			liveOrStaticHandler = liveHandler
		} else {
			log.Printf("⚠️  Live handler failed, using static for pages: %v", err)
		}
	}

	if compiledHandler != nil {
		// Composite handler
		s.routeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if s.isAPIRoute(path) {
				compiledHandler.ServeHTTP(w, r)
				return
			}
			if s.isServerDrivenRoute(path) {
				liveOrStaticHandler.ServeHTTP(w, r)
				return
			}
			if s.isSSRPageRoute(path) {
				compiledHandler.ServeHTTP(w, r)
				return
			}
			staticHandler.ServeHTTP(w, r)
		})
		log.Println("✅ Routes compiled via compiler (API+SSR) and live/static for server-driven")
		return nil
	}

	// Fallback: live handler or simple handler
	if s.liveServer != nil {
		if liveHandler, err := routes.NewLiveHandler("app/routes", s.liveServer, staticHandler); err == nil {
			s.routeHandler = liveHandler
			log.Println("✅ Routes compiled with live protocol support")
			return nil
		}
	}

	if handler, err := routes.NewHandler("app/routes", staticHandler); err == nil {
		s.routeHandler = handler
		log.Println("✅ Routes compiled (simple handler)")
		return nil
	} else {
		return fmt.Errorf("failed to create route handler: %w", err)
	}
}

func (s *devServer) recompileRoutes() error {
	// Re-compile routes after changes
	log.Println("🔄 Recompiling routes...")

	// Refresh classifiers
	_ = s.refreshRouteClassifiers()

	// If we have a compiler, prefer recompile
	if s.routeCompiler != nil {
		if h, err := s.routeCompiler.Recompile(); err == nil && h != nil {
			compiledHandler := h
			staticHandler := http.HandlerFunc(s.serveStatic)
			var liveOrStaticHandler http.Handler = staticHandler
			if s.liveServer != nil {
				if liveHandler, err := routes.NewLiveHandler("app/routes", s.liveServer, staticHandler); err == nil {
					liveOrStaticHandler = liveHandler
				}
			}
			s.routeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				if s.isAPIRoute(path) {
					compiledHandler.ServeHTTP(w, r)
					return
				}
				if s.isServerDrivenRoute(path) {
					liveOrStaticHandler.ServeHTTP(w, r)
					return
				}
				if s.isSSRPageRoute(path) {
					compiledHandler.ServeHTTP(w, r)
					return
				}
				staticHandler.ServeHTTP(w, r)
			})
			log.Println("✅ Routes recompiled via compiler (API+SSR) and live/static for server-driven")
			return nil
		}
		log.Printf("⚠️  Recompile via compiler failed; doing full recompile")
		s.routeCompiler = nil
	}

	// If we have a live handler, try to refresh it
	if liveHandler, ok := s.routeHandler.(*routes.LiveHandler); ok {
		if err := liveHandler.Refresh("app/routes"); err == nil {
			log.Println("✅ Routes refreshed with live support")
			return nil
		}
	}

	// Otherwise do full recompile
	return s.compileRoutes()
}

func (s *devServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check if this is a live protocol connection (path includes /vango/live/)
	if strings.Contains(r.URL.Path, "/vango/live/") {
		// Use the live server for server-driven components
		log.Printf("🔌 Live WebSocket connection: %s", r.URL.Path)
		s.liveServer.HandleWebSocket(w, r)
		return
	}

	// Legacy WebSocket handling for dev server hot reload
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	// Register client
	s.wsMutex.Lock()
	s.wsClients[conn] = true
	s.wsMutex.Unlock()

	defer func() {
		s.wsMutex.Lock()
		delete(s.wsClients, conn)
		s.wsMutex.Unlock()
	}()

	// Handle messages
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle different message types
		switch msg["type"] {
		case "HELLO":
			// Send acknowledgment
			conn.WriteJSON(map[string]interface{}{
				"type": "ACK",
			})
		default:
			log.Printf("Unknown WebSocket message type: %v", msg["type"])
		}
	}
}

func (s *devServer) notifyClients(msgType string, data map[string]interface{}) {
	s.wsMutex.RLock()
	defer s.wsMutex.RUnlock()

	message := map[string]interface{}{
		"type": strings.ToUpper(msgType),
	}
	for k, v := range data {
		message[k] = v
	}

	for client := range s.wsClients {
		if err := client.WriteJSON(message); err != nil {
			log.Printf("Failed to send message to client: %v", err)
		}
	}
}

// refreshRouteClassifiers rescans application routes and classifies them
func (s *devServer) refreshRouteClassifiers() error {
	scanner, err := routes.NewScanner("app/routes")
	if err != nil {
		return err
	}
	routeFiles, err := scanner.ScanRoutes()
	if err != nil {
		return err
	}

	var apiPats, serverPats, ssrPats []string
	for _, rf := range routeFiles {
		if rf.IsAPI {
			apiPats = append(apiPats, rf.URLPattern)
			continue
		}
		if rf.HasServer {
			serverPats = append(serverPats, rf.URLPattern)
			continue
		}
		if rf.HasClient && !rf.HasServer {
			// client-only route, handled by CSR
			continue
		}
		// SSR / universal route
		ssrPats = append(ssrPats, rf.URLPattern)
	}
	s.apiPatterns = apiPats
	s.serverDrivenPatterns = serverPats
	s.ssrPagePatterns = ssrPats
	return nil
}

func (s *devServer) isAPIRoute(path string) bool {
	if strings.HasPrefix(path, "/api/") || path == "/api" {
		return true
	}
	for _, pat := range s.apiPatterns {
		if matchPathRoute(path, pat) {
			return true
		}
	}
	return false
}

func (s *devServer) isServerDrivenRoute(path string) bool {
	for _, pat := range s.serverDrivenPatterns {
		if matchPathRoute(path, pat) {
			return true
		}
	}
	return false
}

func (s *devServer) isSSRPageRoute(path string) bool {
	for _, pat := range s.ssrPagePatterns {
		if matchPathRoute(path, pat) {
			return true
		}
	}
	return false
}

// matchPathRoute supports patterns with :param and *catchall
func matchPathRoute(path, pattern string) bool {
	// Normalize
	if path == pattern || path+"/" == pattern || path == pattern+"/" {
		return true
	}
	pSegs := strings.Split(strings.Trim(pattern, "/"), "/")
	sSegs := strings.Split(strings.Trim(path, "/"), "/")

	// Walk pattern vs path
	i := 0
	j := 0
	for i < len(pSegs) && j < len(sSegs) {
		ps := pSegs[i]
		ss := sSegs[j]

		// Bracket param: [name] or [name:type]
		if strings.HasPrefix(ps, "[") && strings.HasSuffix(ps, "]") {
			inner := ps[1 : len(ps)-1]
			// Catch-all [...name]
			if strings.HasPrefix(inner, "...") {
				// Catch-all consumes the rest
				return true
			}
			// Typed param [name[:type]]
			name := inner
			ptype := "string"
			if k := strings.Index(inner, ":"); k != -1 {
				name = inner[:k]
				ptype = inner[k+1:]
			}
			_ = name // name unused here; matching only
			// Validate based on type
			switch ptype {
			case "int", "int64":
				if len(ss) == 0 {
					return false
				}
				for _, r := range ss {
					if r < '0' || r > '9' {
						return false
					}
				}
			case "uuid":
				if len(ss) != 36 {
					return false
				}
				if ss[8] != '-' || ss[13] != '-' || ss[18] != '-' || ss[23] != '-' {
					return false
				}
			default:
				if ss == "" {
					return false
				}
			}
			i++
			j++
			continue
		}

		// Static segment
		if ps != ss {
			return false
		}
		i++
		j++
	}

	// If pattern has remaining segments, only ok if last was catch-all
	return i == len(pSegs) && j == len(sSegs)
}

func (s *devServer) serveWASM(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/wasm")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, "public/app.wasm")
}

func (s *devServer) serveWasmExec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	// Prefer embedded; fallback to TinyGo if unavailable
	if len(assets.WasmExecJS) > 0 {
		_, _ = w.Write(assets.WasmExecJS)
		return
	}
	output, cmdErr := exec.Command("tinygo", "env", "TINYGOROOT").Output()
	if cmdErr != nil {
		http.Error(w, "Failed to resolve wasm_exec.js", http.StatusInternalServerError)
		return
	}
	wasmExecPath := filepath.Join(strings.TrimSpace(string(output)), "targets/wasm_exec.js")
	content, err := os.ReadFile(wasmExecPath)
	if err != nil {
		http.Error(w, "Failed to load wasm_exec.js", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(content)
}

func (s *devServer) serveBootstrap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	// Serve embedded bootstrap reliably
	content := assets.BootstrapJS
	if len(content) == 0 {
		http.Error(w, "bootstrap.js not embedded", http.StatusInternalServerError)
		return
	}
	// Replace process.env in dev
	contentStr := strings.ReplaceAll(string(content), "process.env.NODE_ENV", "'development'")
	_, _ = w.Write([]byte(contentStr))
}

// findUpwards searches parent directories for the given relative path and returns
// the first absolute path that exists, or empty string if not found.
func findUpwards(rel string) string {
	// Try current and up to 6 parent directories
	dirs := []string{".", "..", "../..", "../../..", "../../../..", "../../../../..", "../../../../../.."}
	for _, d := range dirs {
		p := filepath.Join(d, rel)
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

func (s *devServer) serveStyles(w http.ResponseWriter, r *http.Request) {
	// Remove /styles/ prefix
	path := strings.TrimPrefix(r.URL.Path, "/styles/")

	// Security: prevent directory traversal
	if strings.Contains(path, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Set correct MIME type
	w.Header().Set("Content-Type", "text/css")
	w.Header().Set("Cache-Control", "no-cache")

	// Serve from styles directory
	filePath := filepath.Join("styles", path)
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Write(content)
}

// serveRootStyles serves /styles.css (Tailwind output) with correct MIME type.
// If the file is missing, returns an empty CSS response to avoid console MIME errors.
func (s *devServer) serveRootStyles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	w.Header().Set("Cache-Control", "no-cache")
	content, err := os.ReadFile("public/styles.css")
	if err != nil {
		// Serve empty CSS placeholder
		_, _ = w.Write([]byte("/* vango: styles.css not generated yet */\n"))
		return
	}
	_, _ = w.Write(content)
}

func (s *devServer) serveRouterTable(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	// Serve the generated router table
	content, err := os.ReadFile("router/table.json")
	if err != nil {
		// If file doesn't exist, return empty routes
		w.Write([]byte(`{"routes":[]}`))
		return
	}

	w.Write(content)
}

func (s *devServer) serveStatic(w http.ResponseWriter, r *http.Request) {
	// Default to index.html for root
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Security: prevent directory traversal
	if strings.Contains(path, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Try public directory first
	filePath := filepath.Join("public", strings.TrimPrefix(path, "/"))
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set MIME type based on extension
	ext := filepath.Ext(filePath)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".wasm":
		w.Header().Set("Content-Type", "application/wasm")
	default:
		// Let Go's default MIME type detection handle it
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Write(content)
}

// serveFavicon serves a project favicon if present, otherwise returns 204 to avoid noisy 404.
func (s *devServer) serveFavicon(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat("public/favicon.ico"); err == nil {
		http.ServeFile(w, r, "public/favicon.ico")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Helper functions for build cache

func generateWASMCacheKey() (string, error) {
	// Collect all relevant files for cache key
	files := []string{}

	// Add go.mod and go.sum
	if _, err := os.Stat("go.mod"); err == nil {
		files = append(files, "go.mod")
	}
	if _, err := os.Stat("go.sum"); err == nil {
		files = append(files, "go.sum")
	}

	// Add all Go files in app directory
	goFiles := collectGoFiles("./app")
	files = append(files, goFiles...)

	// Add TinyGo version to cache key
	tinygoVersion := getTinyGoVersion()

	// Generate cache key from files and version
	keyInputs := []string{tinygoVersion}
	for _, file := range files {
		if data, err := os.ReadFile(file); err == nil {
			keyInputs = append(keyInputs, string(data))
		}
	}

	return cache.Key(keyInputs...), nil
}

func collectGoFiles(dir string) []string {
	var files []string

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip vendor and hidden directories
		if info.IsDir() && (strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor") {
			return filepath.SkipDir
		}

		// Collect Go files
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}

		return nil
	})

	return files
}

func getTinyGoVersion() string {
	cmd := exec.Command("tinygo", "version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return string(output)
}
