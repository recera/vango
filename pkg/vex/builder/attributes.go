package builder

// === Form Attributes ===

import "github.com/recera/vango/pkg/vango/vdom"

// Disabled sets the disabled attribute
func (b *ElementBuilder) Disabled(disabled bool) *ElementBuilder {
	if disabled {
		b.props["disabled"] = true
	}
	return b
}

// Required sets the required attribute
func (b *ElementBuilder) Required(required bool) *ElementBuilder {
	if required {
		b.props["required"] = true
	}
	return b
}

// Checked sets the checked attribute
func (b *ElementBuilder) Checked(checked bool) *ElementBuilder {
	if checked {
		b.props["checked"] = true
	}
	return b
}

// ReadOnly sets the readonly attribute
func (b *ElementBuilder) ReadOnly(readonly bool) *ElementBuilder {
	if readonly {
		b.props["readonly"] = true
	}
	return b
}

// Name sets the name attribute
func (b *ElementBuilder) Name(name string) *ElementBuilder {
	b.props["name"] = name
	return b
}

// Value sets the value attribute
func (b *ElementBuilder) Value(value string) *ElementBuilder {
	b.props["value"] = value
	return b
}

// Type sets the type attribute
func (b *ElementBuilder) Type(t string) *ElementBuilder {
	b.props["type"] = t
	return b
}

// Placeholder sets the placeholder attribute
func (b *ElementBuilder) Placeholder(placeholder string) *ElementBuilder {
	b.props["placeholder"] = placeholder
	return b
}

// MaxLength sets the maxlength attribute
func (b *ElementBuilder) MaxLength(length int) *ElementBuilder {
	b.props["maxlength"] = length
	return b
}

// MinLength sets the minlength attribute
func (b *ElementBuilder) MinLength(length int) *ElementBuilder {
	b.props["minlength"] = length
	return b
}

// === Link & Media Attributes ===

// Href sets the href attribute
func (b *ElementBuilder) Href(href string) *ElementBuilder {
	b.props["href"] = href
	return b
}

// Target sets the target attribute
func (b *ElementBuilder) Target(target string) *ElementBuilder {
	b.props["target"] = target
	return b
}

// Rel sets the rel attribute
func (b *ElementBuilder) Rel(rel string) *ElementBuilder {
	b.props["rel"] = rel
	return b
}

// Src sets the src attribute
func (b *ElementBuilder) Src(src string) *ElementBuilder {
	b.props["src"] = src
	return b
}

// Alt sets the alt attribute
func (b *ElementBuilder) Alt(alt string) *ElementBuilder {
	b.props["alt"] = alt
	return b
}

// Width sets the width attribute
func (b *ElementBuilder) Width(width string) *ElementBuilder {
	b.props["width"] = width
	return b
}

// Height sets the height attribute
func (b *ElementBuilder) Height(height string) *ElementBuilder {
	b.props["height"] = height
	return b
}

// Loading sets the loading attribute (lazy, eager)
func (b *ElementBuilder) Loading(loading string) *ElementBuilder {
	b.props["loading"] = loading
	return b
}

// === Table Attributes ===

// Colspan sets the colspan attribute
func (b *ElementBuilder) Colspan(span int) *ElementBuilder {
	b.props["colspan"] = span
	return b
}

// Rowspan sets the rowspan attribute
func (b *ElementBuilder) Rowspan(span int) *ElementBuilder {
	b.props["rowspan"] = span
	return b
}

// === List Attributes ===

// Start sets the start attribute for ordered lists
func (b *ElementBuilder) Start(start int) *ElementBuilder {
	b.props["start"] = start
	return b
}

// Reversed sets the reversed attribute for ordered lists
func (b *ElementBuilder) Reversed(reversed bool) *ElementBuilder {
	if reversed {
		b.props["reversed"] = true
	}
	return b
}

// === Form Control Attributes ===

// Autocomplete sets the autocomplete attribute
func (b *ElementBuilder) Autocomplete(autocomplete string) *ElementBuilder {
	b.props["autocomplete"] = autocomplete
	return b
}

// Autofocus sets the autofocus attribute
func (b *ElementBuilder) Autofocus(autofocus bool) *ElementBuilder {
	if autofocus {
		b.props["autofocus"] = true
	}
	return b
}

// AutoFocus sets the autofocus attribute (alias)
func (b *ElementBuilder) AutoFocus(autofocus bool) *ElementBuilder {
	return b.Autofocus(autofocus)
}

// Multiple sets the multiple attribute
func (b *ElementBuilder) Multiple(multiple bool) *ElementBuilder {
	if multiple {
		b.props["multiple"] = true
	}
	return b
}

// Selected sets the selected attribute (for option elements)
func (b *ElementBuilder) Selected(selected bool) *ElementBuilder {
	if selected {
		b.props["selected"] = true
	}
	return b
}

// Pattern sets the pattern attribute
func (b *ElementBuilder) Pattern(pattern string) *ElementBuilder {
	b.props["pattern"] = pattern
	return b
}

// Min sets the min attribute
func (b *ElementBuilder) Min(min string) *ElementBuilder {
	b.props["min"] = min
	return b
}

// Max sets the max attribute
func (b *ElementBuilder) Max(max string) *ElementBuilder {
	b.props["max"] = max
	return b
}

// Step sets the step attribute
func (b *ElementBuilder) Step(step string) *ElementBuilder {
	b.props["step"] = step
	return b
}

// === Textarea Attributes ===

// Rows sets the rows attribute
func (b *ElementBuilder) Rows(rows int) *ElementBuilder {
	b.props["rows"] = rows
	return b
}

// Cols sets the cols attribute
func (b *ElementBuilder) Cols(cols int) *ElementBuilder {
	b.props["cols"] = cols
	return b
}

// Wrap sets the wrap attribute
func (b *ElementBuilder) Wrap(wrap string) *ElementBuilder {
	b.props["wrap"] = wrap
	return b
}

// === Data Attributes ===

// Data sets a data attribute
func (b *ElementBuilder) Data(key, value string) *ElementBuilder {
	b.props["data-"+key] = value
	return b
}

// === Additional Event Handlers ===

// OnFocus sets the onfocus handler
func (b *ElementBuilder) OnFocus(handler func()) *ElementBuilder {
	b.props["onfocus"] = handler
	return b
}

// OnBlur sets the onblur handler
func (b *ElementBuilder) OnBlur(handler func()) *ElementBuilder {
	b.props["onblur"] = handler
	return b
}

// OnKeyDown sets the onkeydown handler
func (b *ElementBuilder) OnKeyDown(handler interface{}) *ElementBuilder {
	b.props["onkeydown"] = handler
	return b
}

// OnKeyUp sets the onkeyup handler
func (b *ElementBuilder) OnKeyUp(handler interface{}) *ElementBuilder {
	b.props["onkeyup"] = handler
	return b
}

// OnKeyPress sets the onkeypress handler
func (b *ElementBuilder) OnKeyPress(handler interface{}) *ElementBuilder {
	b.props["onkeypress"] = handler
	return b
}

// OnMouseOver sets the onmouseover handler
func (b *ElementBuilder) OnMouseOver(handler interface{}) *ElementBuilder {
	b.props["onmouseover"] = handler
	return b
}

// OnMouseOut sets the onmouseout handler
func (b *ElementBuilder) OnMouseOut(handler interface{}) *ElementBuilder {
	b.props["onmouseout"] = handler
	return b
}

// OnMouseDown sets the onmousedown handler
func (b *ElementBuilder) OnMouseDown(handler interface{}) *ElementBuilder {
	b.props["onmousedown"] = handler
	return b
}

// OnMouseUp sets the onmouseup handler
func (b *ElementBuilder) OnMouseUp(handler interface{}) *ElementBuilder {
	b.props["onmouseup"] = handler
	return b
}

// OnMouseMove sets the onmousemove handler
func (b *ElementBuilder) OnMouseMove(handler interface{}) *ElementBuilder {
	b.props["onmousemove"] = handler
	return b
}

// OnWheel sets the onwheel handler (for zoom)
func (b *ElementBuilder) OnWheel(handler interface{}) *ElementBuilder {
	b.props["onwheel"] = handler
	return b
}

// OnDblClick sets the ondblclick handler
func (b *ElementBuilder) OnDblClick(handler interface{}) *ElementBuilder {
    b.props["ondblclick"] = handler
    return b
}

// === Label Attributes ===

// For sets the for attribute (for labels)
func (b *ElementBuilder) For(forID string) *ElementBuilder {
	b.props["for"] = forID
	return b
}

// === Custom Attributes ===

// Attr sets a custom attribute
func (b *ElementBuilder) Attr(key string, value interface{}) *ElementBuilder {
	b.props[key] = value
	return b
}

// === Refs ===

// Ref sets a callback that receives the underlying DOM element (js.Value) after creation/hydration
func (b *ElementBuilder) Ref(ref func(vdom.ElementRef)) *ElementBuilder {
	if ref != nil {
		b.props["ref"] = ref
	}
	return b
}
