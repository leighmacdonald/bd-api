package main

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/leighmacdonald/bd-api/domain"
)

type styleEncoder struct {
	style     *chroma.Style
	formatter *html.Formatter
	lexer     chroma.Lexer
}

func newStyleEncoder() *styleEncoder {
	newStyle := styles.Get("monokai")
	if newStyle == nil {
		newStyle = styles.Fallback
	}

	return &styleEncoder{
		style:     newStyle,
		formatter: html.New(html.WithClasses(true)),
		lexer:     lexers.Get("json"),
	}
}

func (s *styleEncoder) Encode(value any) (string, string, error) {
	var jsonBody bytes.Buffer
	jsonEncoder := json.NewEncoder(&jsonBody)
	jsonEncoder.SetIndent("", "    ")
	if errJSON := jsonEncoder.Encode(value); errJSON != nil {
		return "", "", errors.Join(errJSON, domain.ErrResponseJSON)
	}

	iterator, errTokenize := s.lexer.Tokenise(&chroma.TokeniseOptions{State: "root", EnsureLF: true}, jsonBody.String())
	if errTokenize != nil {
		return "", "", errors.Join(errTokenize, domain.ErrResponseTokenize)
	}

	cssBuf := bytes.NewBuffer(nil)
	if err := s.formatter.WriteCSS(cssBuf, s.style); err != nil {
		return "", "", errors.Join(err, domain.ErrResponseCSS)
	}

	bodyBuf := bytes.NewBuffer(nil)
	if errFormat := s.formatter.Format(bodyBuf, s.style, iterator); errFormat != nil {
		return "", "", errors.Join(errFormat, domain.ErrResponseFormat)
	}

	return cssBuf.String(), bodyBuf.String(), nil
}
