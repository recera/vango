package styling

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ComponentStyle represents a component's scoped styles
// At runtime before the build rewrite, Style() simply returns this struct
// The build step replaces the call expression with a literal containing hashed names
type ComponentStyle struct {
	// Hash is assigned by build step, blank in dev until first compile
	Hash string
	
	// names maps original class names to hashed class names
	// e.g., "card" -> "_v1a2b_card"
	names map[string]string
	
	// CSS contains the actual stylesheet content
	// In dev mode, this contains the original CSS
	// In production, this would be extracted at build time
	CSS string
}

// Style creates a new ComponentStyle
// During development, this returns a ComponentStyle with hashed names
// The build process will replace this call with a literal struct
func Style(css string) *ComponentStyle {
	// Generate hash from CSS content
	h := sha256.New()
	h.Write([]byte(css))
	hashBytes := h.Sum(nil)
	hash := "_" + hex.EncodeToString(hashBytes)[:6] // Use first 6 chars for brevity
	
	// Parse the CSS to extract class names
	classNames := extractClassNames(css)
	nameMap := make(map[string]string)
	
	// Generate hashed class names for scoping
	for _, className := range classNames {
		// Create hashed version: _hash_originalName
		hashedName := hash + "_" + strings.ReplaceAll(className, ".", "_")
		hashedName = strings.ReplaceAll(hashedName, ":", "_")
		nameMap[className] = hashedName
	}
	
	return &ComponentStyle{
		Hash:  hash,
		names: nameMap,
		CSS:   css,
	}
}

// extractClassNames extracts class names from CSS
func extractClassNames(css string) []string {
	classMap := make(map[string]bool)
	
	// Remove comments first
	css = removeComments(css)
	
	// Split by rules (roughly)
	// Look for patterns like .className followed by {, space, comma, colon, or [
	i := 0
	for i < len(css) {
		// Find next period
		if css[i] == '.' {
			start := i + 1
			end := start
			
			// Find the end of the class name
			for end < len(css) {
				c := css[end]
				if c == ' ' || c == '{' || c == ',' || c == ':' || c == '[' || 
				   c == '\n' || c == '\r' || c == '\t' || c == '.' || c == '#' ||
				   c == '>' || c == '+' || c == '~' || c == '(' {
					break
				}
				end++
			}
			
			if end > start {
				className := css[start:end]
				
				// Check for compound selectors like .card.active
				if end < len(css) && css[end] == '.' {
					// Continue to get the full compound class
					compoundStart := start
					for end < len(css) && css[end] == '.' {
						end++ // Skip the dot
						for end < len(css) {
							c := css[end]
							if c == ' ' || c == '{' || c == ',' || c == ':' || c == '[' ||
							   c == '\n' || c == '\r' || c == '\t' || c == '.' || c == '#' ||
							   c == '>' || c == '+' || c == '~' || c == '(' {
								break
							}
							end++
						}
					}
					// Get the full compound selector
					fullSelector := css[compoundStart:end]
					classMap[fullSelector] = true
				} else if end < len(css) && css[end] == ':' {
					// Handle pseudo-class like .card:hover
					pseudoEnd := end + 1
					for pseudoEnd < len(css) {
						c := css[pseudoEnd]
						if c == ' ' || c == '{' || c == ',' || c == '[' ||
						   c == '\n' || c == '\r' || c == '\t' || c == '(' {
							break
						}
						pseudoEnd++
					}
					fullSelector := css[start:pseudoEnd]
					classMap[fullSelector] = true
				} else {
					// Regular single class
					classMap[className] = true
				}
			}
			i = end
		} else {
			i++
		}
	}
	
	// Convert map to slice
	classes := make([]string, 0, len(classMap))
	for class := range classMap {
		classes = append(classes, class)
	}
	
	return classes
}

// removeComments removes CSS comments from the string
func removeComments(css string) string {
	result := strings.Builder{}
	i := 0
	for i < len(css) {
		if i < len(css)-1 && css[i] == '/' && css[i+1] == '*' {
			// Find end of comment
			i += 2
			for i < len(css)-1 {
				if css[i] == '*' && css[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
		} else {
			result.WriteByte(css[i])
			i++
		}
	}
	return result.String()
}

// Class returns the hashed class name for the given original name
// Falls back to the original name during development
func (c *ComponentStyle) Class(name string) string {
	if c == nil {
		return name
	}
	
	if v, ok := c.names[name]; ok {
		return v
	}
	
	// Fallback to original name during dev-first-compile to avoid nils
	return name
}

// Classes returns multiple hashed class names separated by space
func (c *ComponentStyle) Classes(names ...string) string {
	result := ""
	for i, name := range names {
		if i > 0 {
			result += " "
		}
		result += c.Class(name)
	}
	return result
}

// Has returns whether a class name exists in this component's styles
func (c *ComponentStyle) Has(name string) bool {
	if c == nil || c.names == nil {
		return false
	}
	_, ok := c.names[name]
	return ok
}

// GetHash returns the hash for this component's styles
// Used by the build system to generate unique CSS file names
func (c *ComponentStyle) GetHash() string {
	if c == nil {
		return ""
	}
	return c.Hash
}