package service

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

type ExtractedTextDocument struct {
	Title  string
	Author string
	Text   string
}

// ExtractTextDocument extracts plain text (+ best-effort metadata) from supported non-PDF formats.
// Supported formats: "txt", "md", "epub".
func ExtractTextDocument(format string, originalName string, fileBytes []byte) (ExtractedTextDocument, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "txt", "md":
		ext := filepath.Ext(originalName)
		title := strings.TrimSpace(strings.TrimSuffix(originalName, ext))
		text := strings.TrimSpace(string(bytes.ToValidUTF8(fileBytes, []byte{})))
		return ExtractedTextDocument{
			Title:  title,
			Author: "",
			Text:   text,
		}, nil
	case "epub":
		return extractEPUB(fileBytes, originalName)
	default:
		return ExtractedTextDocument{}, fmt.Errorf("unsupported format: %s", format)
	}
}

// BuildTextBlocksFromText converts extracted text into TextBlocks and pseudo-pages.
// Returns (blocks, pageCount, wordCount).
func BuildTextBlocksFromText(p *PDFProcessor, text string) ([]TextBlock, int, int) {
	const maxPageChars = 2600

	if p == nil {
		// Should never happen in production; keep safe defaults.
		text = strings.TrimSpace(text)
		if text == "" {
			return []TextBlock{{
				Type:       "paragraph",
				Content:    "",
				Level:      0,
				PageNumber: 1,
				Position:   0,
			}}, 1, 0
		}
		return []TextBlock{{
			Type:       "paragraph",
			Content:    text,
			Level:      0,
			PageNumber: 1,
			Position:   0,
		}}, 1, len(strings.Fields(text))
	}

	normalized := strings.TrimSpace(text)
	wordCount := len(strings.Fields(normalized))

	if normalized == "" {
		return []TextBlock{{
			Type:       "paragraph",
			Content:    "",
			Level:      0,
			PageNumber: 1,
			Position:   0,
		}}, 1, 0
	}

	paragraphs := p.splitIntoParagraphs(normalized)
	pages := paginateParagraphs(p, paragraphs, maxPageChars)
	if len(pages) == 0 {
		pages = []string{""}
	}

	blocks := make([]TextBlock, 0, len(paragraphs))
	for i, pageText := range pages {
		pageNumber := i + 1
		pos := 0

		paras := strings.Split(pageText, "\n\n")
		for _, para := range paras {
			para = strings.TrimSpace(para)
			if para == "" {
				continue
			}
			blockType := "paragraph"
			level := 0
			if p.isHeading(para) {
				blockType = "heading"
				level = 1
			}
			para = p.sanitizeText(para)
			blocks = append(blocks, TextBlock{
				Type:       blockType,
				Content:    para,
				Level:      level,
				PageNumber: pageNumber,
				Position:   pos,
			})
			pos++
		}

		// Preserve at least one block per page for structure.
		if pos == 0 {
			blocks = append(blocks, TextBlock{
				Type:       "paragraph",
				Content:    "",
				Level:      0,
				PageNumber: pageNumber,
				Position:   0,
			})
		}
	}

	return blocks, len(pages), wordCount
}

func paginateParagraphs(p *PDFProcessor, paragraphs []string, maxChars int) []string {
	var pages []string
	var sb strings.Builder

	flush := func() {
		page := strings.TrimSpace(sb.String())
		pages = append(pages, page)
		sb.Reset()
	}

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		para = p.sanitizeText(para)
		if para == "" {
			continue
		}

		// If a single paragraph is longer than a page, still put it on its own page.
		if sb.Len() == 0 && len(para) > maxChars {
			pages = append(pages, strings.TrimSpace(para))
			continue
		}

		// If adding this paragraph would overflow, start a new page.
		if sb.Len() > 0 && sb.Len()+2+len(para) > maxChars {
			flush()
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(para)
	}

	if sb.Len() > 0 {
		flush()
	}

	return pages
}

// --- EPUB extraction ---

func extractEPUB(epubBytes []byte, originalName string) (ExtractedTextDocument, error) {
	zr, err := zip.NewReader(bytes.NewReader(epubBytes), int64(len(epubBytes)))
	if err != nil {
		return ExtractedTextDocument{}, fmt.Errorf("failed to open epub: %w", err)
	}

	containerBytes, err := readZipFile(zr, "META-INF/container.xml")
	if err != nil {
		return ExtractedTextDocument{}, fmt.Errorf("invalid epub (missing container.xml): %w", err)
	}

	opfPath, err := findOPFPath(containerBytes)
	if err != nil || strings.TrimSpace(opfPath) == "" {
		return ExtractedTextDocument{}, fmt.Errorf("invalid epub (missing package path)")
	}

	opfBytes, err := readZipFile(zr, opfPath)
	if err != nil {
		return ExtractedTextDocument{}, fmt.Errorf("invalid epub (missing package file): %w", err)
	}

	title, author, orderedHrefs := parseOPF(opfBytes)
	if title == "" {
		ext := filepath.Ext(originalName)
		title = strings.TrimSpace(strings.TrimSuffix(originalName, ext))
	}

	// Resolve spine hrefs relative to OPF directory.
	opfDir := path.Dir(opfPath)
	if opfDir == "." {
		opfDir = ""
	}

	var chapters []string
	chapters = make([]string, 0, len(orderedHrefs))
	for _, href := range orderedHrefs {
		href = strings.TrimSpace(href)
		if href == "" {
			continue
		}
		unescaped, _ := url.PathUnescape(href)
		if unescaped != "" {
			href = unescaped
		}
		full := path.Clean(path.Join(opfDir, href))
		b, err := readZipFile(zr, full)
		if err != nil {
			// Best-effort: skip missing items.
			continue
		}
		t := htmlToText(b)
		t = normalizeText(t)
		if t != "" {
			chapters = append(chapters, t)
		}
	}

	combined := strings.TrimSpace(strings.Join(chapters, "\n\n"))
	return ExtractedTextDocument{
		Title:  strings.TrimSpace(title),
		Author: strings.TrimSpace(author),
		Text:   combined,
	}, nil
}

func readZipFile(zr *zip.Reader, name string) ([]byte, error) {
	// Try exact match first.
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	// Then case-insensitive match.
	lower := strings.ToLower(name)
	for _, f := range zr.File {
		if strings.ToLower(f.Name) == lower {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

func findOPFPath(containerXML []byte) (string, error) {
	// container.xml is usually:
	// <container ...><rootfiles><rootfile full-path="OEBPS/content.opf" .../></rootfiles></container>
	type rootfile struct {
		FullPath string `xml:"full-path,attr"`
	}
	type rootfiles struct {
		Rootfiles []rootfile `xml:"rootfile"`
	}
	type container struct {
		Rootfiles rootfiles `xml:"rootfiles"`
	}

	var c container
	if err := xml.Unmarshal(containerXML, &c); err != nil {
		return "", err
	}
	for _, rf := range c.Rootfiles.Rootfiles {
		if strings.TrimSpace(rf.FullPath) != "" {
			return strings.TrimSpace(rf.FullPath), nil
		}
	}
	return "", fmt.Errorf("rootfile not found")
}

func parseOPF(opf []byte) (title string, author string, spineHrefs []string) {
	// Parse metadata + manifest + spine with namespace-agnostic matching using Name.Local.
	type manifestItem struct {
		ID   string
		Href string
	}
	manifest := map[string]manifestItem{}
	spineIDs := make([]string, 0, 64)

	dec := xml.NewDecoder(bytes.NewReader(opf))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch strings.ToLower(se.Name.Local) {
		case "title":
			if title == "" {
				title = strings.TrimSpace(readElementText(dec))
			}
		case "creator":
			if author == "" {
				author = strings.TrimSpace(readElementText(dec))
			}
		case "item":
			var id, href string
			for _, a := range se.Attr {
				switch strings.ToLower(a.Name.Local) {
				case "id":
					id = a.Value
				case "href":
					href = a.Value
				}
			}
			if id != "" && href != "" {
				manifest[id] = manifestItem{ID: id, Href: href}
			}
		case "itemref":
			var idref string
			for _, a := range se.Attr {
				if strings.ToLower(a.Name.Local) == "idref" {
					idref = a.Value
					break
				}
			}
			if idref != "" {
				spineIDs = append(spineIDs, idref)
			}
		}
	}

	spineHrefs = make([]string, 0, len(spineIDs))
	for _, id := range spineIDs {
		if item, ok := manifest[id]; ok && item.Href != "" {
			spineHrefs = append(spineHrefs, item.Href)
		}
	}
	return title, author, spineHrefs
}

func readElementText(dec *xml.Decoder) string {
	var out strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.CharData:
			out.Write([]byte(t))
		case xml.EndElement:
			return out.String()
		}
	}
	return out.String()
}

func htmlToText(b []byte) string {
	doc, err := html.Parse(bytes.NewReader(b))
	if err != nil || doc == nil {
		return ""
	}

	block := map[string]bool{
		"p": true, "div": true, "section": true, "article": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"li": true, "ul": true, "ol": true, "blockquote": true,
	}
	skip := map[string]bool{
		"script": true, "style": true, "head": true, "title": true, "nav": true,
	}

	var sb strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n == nil {
			return
		}
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			if skip[tag] {
				return
			}
			if tag == "br" {
				sb.WriteString("\n")
			}
			if block[tag] {
				sb.WriteString("\n\n")
			}
		}
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") && !strings.HasSuffix(sb.String(), " ") {
					sb.WriteString(" ")
				}
				sb.WriteString(t)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			if block[tag] {
				sb.WriteString("\n\n")
			}
		}
	}
	walk(doc)

	return sb.String()
}

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// Replace non-breaking spaces.
	s = strings.ReplaceAll(s, "\u00a0", " ")

	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	blank := 0
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			blank++
			if blank <= 2 {
				out = append(out, "")
			}
			continue
		}
		blank = 0
		out = append(out, t)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

