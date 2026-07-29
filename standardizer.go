// Copyright (c) 2023 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hujson

import (
	"io"
	"slices"
	"unicode/utf8"
)

// NewStandardizer returns an [io.Reader] that strips HuJSON-specific features
// by replacing all non-whitespace characters in comments and trailing commas
// with space characters, thus preserving original line numbers and byte offsets.
//
// Unlike [Standardize], this does not validate the complete HuJSON grammar but
// does minimal transformation to convert valid HuJSON into valid standard JSON.
// It relies on a standard JSON parser to later detect syntax errors.
// The output is valid JSON if and only if the input is valid HuJSON.
func NewStandardizer(rd io.Reader) *Standardizer {
	return &Standardizer{rd: rd}
}

// Standardizer is an [io.Reader] that reads standard JSON from a HuJSON stream.
type Standardizer struct {
	rd    io.Reader
	rdErr error // non-persistent read error
	standardizerBuffer
}

// Reset discards the Standardizer's state and
// makes it equivalent to calling NewReader with rd instead.
func (r *Standardizer) Reset(rd io.Reader) {
	*r = Standardizer{rd: rd, standardizerBuffer: standardizerBuffer{buffer: r.buffer[:0]}}
}

// Read implements [io.Reader], reading standardized JSON
// from the underlying stream of HuJSON input.
func (r *Standardizer) Read(b []byte) (n int, err error) {
	defer func() {
		// Only report errors if there is no more standardize JSON data to copy.
		if r.jsonOffset >= r.commaOffset {
			err = r.rdErr
			r.rdErr = nil // let underlying io.Reader handle persistence
			if err == io.EOF && r.expectingMore() {
				err = io.ErrUnexpectedEOF
			}
		}
	}()
	switch {
	// Check whether there is already standardized data to copy out.
	case r.jsonOffset < r.commaOffset:
		n = copy(b, r.buffer[r.jsonOffset:r.commaOffset])
		r.jsonOffset += n
	// Check whether we encountered a previous read error.
	case r.rdErr != nil:
		break
	// Check whether we already have data in the internal buffer.
	// If so, read into it and convert it within there,
	// copying out any standardized JSON data.
	case r.commaOffset < len(r.buffer):
		n = len(r.buffer) - r.commaOffset
		r.buffer = slices.Grow(r.buffer, n)
		n, r.rdErr = r.rd.Read(r.buffer[len(r.buffer):][:n])
		r.buffer = r.buffer[:len(r.buffer)+n]
		r.standardize()
		n = copy(b, r.buffer[r.jsonOffset:r.commaOffset])
		r.jsonOffset += n
	// Otherwise, the internal buffer is empty. As an optimization,
	// read directly into the external buffer and standardize it in place
	// using the previous state. Some data may not be standardized,
	// so the unconverted data and state must be preserved.
	default:
		n, r.rdErr = r.rd.Read(b)
		sb := standardizerBuffer{buffer: b[:n], state: r.state}
		sb.standardize()
		r.standardizerBuffer = standardizerBuffer{
			buffer:       append(r.buffer[:0], b[sb.commaOffset:n]...),
			hujsonOffset: sb.hujsonOffset - sb.commaOffset,
			state:        sb.state,
		}
		n = sb.commaOffset
	}
	return n, err
}

// standardizerBuffer is a buffer split into several segments:
//
//   - buffer[:jsonOffset] contains data that has already been copied out
//     to a Read call. This is considered unused buffer space.
//
//   - buffer[jsonOffset:commaOffset] contains already standardized data
//     that is safe to copy out to a future Read call.
//
//   - buffer[commaOffset:hujsonOffset] contains data that is standardized except
//     that it may start with a trailing comma, which cannot be standardized
//     until we find the next JSON token. If the next token is a closing
//     object or array delimiter, then we must elide the comma.
//     Since there is no Limit to the whitespace between a trailing comma and
//     the closing delimiter, this may buffer an unbounded amount of memory.
//     This is always empty unless commaState == afterPossibleTrailingComma.
//
//   - buffer[hujsonOffset:] contains HuJSON data that is not yet standardized.
//     After calling standardize, this is either empty or contains a
//     short fragment because the meaning cannot yet be determined.
//     Fragments can occur within a comment when validating for UTF-8 and
//     a UTF-8 encoded sequence is truncated.
//     It can also occur within a block comment where the buffer
//     ends with a '*' and we do not know if the next character is a '/' or not,
//     which may terminate or continue the block comment sequence.
//     A fragment is always shorter than [utf8.UTFMax].
//
// Invariant: 0 <= jsonOffset <= commaOffset <= hujsonOffset <= len(buffer)
//
// It maintains a finite state machine for eliding comments and trailing commas.
// It does not validate for JSON as that requires a push-down automaton,
// which requires O(n) of stack memory.
type standardizerBuffer struct {
	buffer       []byte
	jsonOffset   int
	commaOffset  int
	hujsonOffset int

	state struct {
		comment commentState
		comma   commaState
	}
}

// commentState is a finite state machine for eliding HuJSON comments.
type commentState uint8

const (
	withinWhitespace       commentState = iota // zero or more whitespace characters
	withinLineComment                          // begins with "//" and ends with "\n"
	withinBlockComment                         // begins with "/*" and ends with "*/"
	withinStringLiteral                        // begins with '"' and ends with unescaped '"'
	withinNonStringLiteral                     // one or more non-whitespace or non-structural characters
)

// commaState is a finite state machine for eliding HuJSON trailing commas.
// A trailing comma only occurs after the completion of a JSON value and
// before a closing object or array delimiter.
type commaState uint8

const (
	beforeValueEnd commaState = iota
	afterValueEnd
	afterPossibleTrailingComma
)

// standardize standardizes HuJSON as standard JSON.
// This is an idempotent operation.
func (s *standardizerBuffer) standardize() {
	b := s.buffer
	i := s.hujsonOffset
stateMachine: // whenever state changes, continue here
	for uint(i) < uint(len(b)) {
		switch s.state.comment {
		case withinWhitespace: // JSON whitespace
			for uint(i) < uint(len(b)) {
				switch b[i] {
				case ' ', '\n', '\r', '\t': // skip over whitespace
					i += len(" ")
				case '/': // possible comment
					if uint(i+1) >= uint(len(b)) {
						break stateMachine // truncated input
					}
					switch b[i+1] {
					case '/': // HuJSON line comment
						copy(b[i:], "  ")
						s.state.comment = withinLineComment
						i += len("//")
						continue stateMachine
					case '*': // HuJSON block comment
						copy(b[i:], "  ")
						s.state.comment = withinBlockComment
						i += len("/*")
						continue stateMachine
					default: // invalid token; see withinNonStringLiteral case below
						s.state.comment = withinNonStringLiteral
						s.state.comma = beforeValueEnd
						i += len("/")
						continue stateMachine
					}
				case '{', '[', ':':
					s.state.comma = beforeValueEnd
					i += len("{")
					continue stateMachine
				case ',':
					if s.state.comma == afterValueEnd {
						s.state.comma = afterPossibleTrailingComma
						s.commaOffset = i
					} else {
						s.state.comma = beforeValueEnd
					}
					i += len(",")
					continue stateMachine
				case '}', ']':
					if s.state.comma == afterPossibleTrailingComma {
						b[s.commaOffset] = ' '
					}
					s.state.comma = afterValueEnd
					i += len("}")
					continue stateMachine
				case '"':
					s.state.comment = withinStringLiteral
					s.state.comma = beforeValueEnd
					i += len(`"`)
					continue stateMachine
				default:
					s.state.comment = withinNonStringLiteral
					s.state.comma = beforeValueEnd
					i += len(`?`)
					continue stateMachine
				}
			}
		case withinLineComment, withinBlockComment: // HuJSON comments
			for uint(i) < uint(len(b)) {
				switch {
				case b[i] == '\n' && s.state.comment == withinLineComment:
					i += len("\n")
					s.state.comment = withinWhitespace
					continue stateMachine
				case b[i] == '*' && s.state.comment == withinBlockComment:
					if uint(i+1) >= uint(len(b)) {
						break stateMachine // truncated input
					}
					if b[i+1] == '/' {
						copy(b[i:], "  ")
						i += len("*/")
						s.state.comment = withinWhitespace
						continue stateMachine
					}
					fallthrough
				case b[i] < utf8.RuneSelf: // single-byte ASCII
					switch b[i] {
					case ' ', '\n', '\r', '\t':
					default:
						b[i] = ' ' // convert non-whitespace to space
					}
					i += len(" ")
				default: // multi-byte Unicode
					// Invalid UTF-8 bytes are not replaced with spaces so that
					// a standard JSON parser can detect them as invalid syntax.
					r, rn := utf8.DecodeRune(b[i:])
					switch {
					case r != utf8.RuneError || rn != 1:
						copy(b[i:][:rn], "    ") // replace valid UTF-8 with space characters
					case !utf8.FullRune(b[i:]):
						break stateMachine // truncated UTF-8 sequence
					}
					i += rn
				}
			}
		case withinStringLiteral: // JSON strings
			for uint(i) < uint(len(b)) {
				switch b[i] {
				case '"': // terminating double quote
					s.state.comment = withinWhitespace
					s.state.comma = afterValueEnd
					i += len(`"`)
					continue stateMachine
				case '\\': // escaped byte (possibly a double quote)
					if uint(i+1) >= uint(len(b)) {
						break stateMachine // truncated input
					}
					i += len(`\?`)
				default: // non-escaped byte
					i += len("?")
				}
			}
		case withinNonStringLiteral: // JSON null, booleans, numbers
			// This treats all non-whitespace and non-structural characters as
			// part of a JSON non-string literal. This may include invalid JSON,
			// which is admissible since it will be passed on verbatim for
			// a standard JSON parser to eventually reject as a syntax error.
			for uint(i) < uint(len(b)) {
				switch b[i] {
				case ' ', '\n', '\r', '\t', '/', '{', '[', ':', ',', '}', ']', '"':
					s.state.comment = withinWhitespace
					s.state.comma = afterValueEnd
					continue stateMachine
				default:
					i += len("?")
				}
			}
		}
	}
	if s.state.comma != afterPossibleTrailingComma {
		s.commaOffset = i
	}
	s.hujsonOffset = i
	s.buffer = b
}

// expectingMore reports whether there might be more standard JSON data to read.
func (s *standardizerBuffer) expectingMore() bool {
	return s.commaOffset < len(s.buffer) || s.state.comment == withinLineComment || s.state.comment == withinBlockComment
}
