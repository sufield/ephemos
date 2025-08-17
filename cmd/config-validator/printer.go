package main

import (
	"encoding/json"
	"fmt"
	"io"
)

// Printer handles formatted output with emoji support
type Printer struct {
	out   io.Writer
	err   io.Writer
	emoji bool
	quiet bool
}

// NewPrinter creates a new printer with the specified configuration
func NewPrinter(out, err io.Writer, emoji, quiet bool) *Printer {
	return &Printer{
		out:   out,
		err:   err,
		emoji: emoji,
		quiet: quiet,
	}
}

// Success prints a success message
func (p *Printer) Success(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "✅", msg)
}

// Info prints an informational message
func (p *Printer) Info(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "🔍", msg)
}

// Infof prints a formatted informational message
func (p *Printer) Infof(format string, args ...interface{}) {
	if p.quiet {
		return
	}
	p.Info(fmt.Sprintf(format, args...))
}

// Lock prints a security-related message
func (p *Printer) Lock(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "🔒", msg)
}

// Warn prints a warning message
func (p *Printer) Warn(msg string) {
	p.line(p.err, "⚠️", msg)
}

// Error prints an error message
func (p *Printer) Error(msg string) {
	p.line(p.err, "❌", msg)
}

// Errorf prints a formatted error message
func (p *Printer) Errorf(format string, args ...interface{}) {
	p.Error(fmt.Sprintf(format, args...))
}

// Plain prints a message without emoji
func (p *Printer) Plain(msg string) {
	if p.quiet {
		return
	}
	fmt.Fprintln(p.out, msg)
}

// Tip prints a tip message
func (p *Printer) Tip(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "💡", msg)
}

// Banner prints a banner message
func (p *Printer) Banner(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "🎉", msg)
}

// Production prints a production-related message
func (p *Printer) Production(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "🏭", msg)
}

// Section prints a section header
func (p *Printer) Section(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "📋", msg)
}

// File prints a file-related message
func (p *Printer) File(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "📁", msg)
}

// Cycle prints a cycle/refresh message
func (p *Printer) Cycle(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "🔄", msg)
}

// Book prints a documentation-related message
func (p *Printer) Book(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "📚", msg)
}

// Shield prints a security shield message
func (p *Printer) Shield(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "🛡️", msg)
}

// Key prints a key/credential message
func (p *Printer) Key(msg string) {
	if p.quiet {
		return
	}
	p.line(p.out, "🔐", msg)
}

// Bullet prints a bullet point
func (p *Printer) Bullet(msg string) {
	if p.quiet {
		return
	}
	if p.emoji {
		fmt.Fprintf(p.out, "  • %s\n", msg)
	} else {
		fmt.Fprintf(p.out, "  - %s\n", msg)
	}
}

// line prints a line with optional emoji prefix
func (p *Printer) line(w io.Writer, emoji, msg string) {
	if p.emoji {
		fmt.Fprintf(w, "%s %s\n", emoji, msg)
	} else {
		fmt.Fprintln(w, msg)
	}
}

// Newline prints a newline
func (p *Printer) Newline() {
	if !p.quiet {
		fmt.Fprintln(p.out)
	}
}

// PrintJSON prints data as JSON
func (p *Printer) PrintJSON(data interface{}) error {
	encoder := json.NewEncoder(p.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
