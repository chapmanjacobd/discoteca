package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/textutil"
)

func (c *Server) HandleOPDS(w http.ResponseWriter, r *http.Request) {
	flags := c.GetGlobalFlags()
	flags.TextOnly = true
	flags.All = true

	media, err := query.MediaQuery(r.Context(), c.Databases, flags)
	if err != nil {
		logging.Log.Error("OPDS query failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/atom+xml;charset=utf-8")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:opds="http://opds-spec.org/2010/catalog">
  <id>discoteca-text</id>
  <title>Discoteca Text</title>
  <updated>`+time.Now().Format(time.RFC3339)+`</updated>
  <author><name>Discoteca</name></author>
`)

	host := r.Host
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	basePath := BasePathFromContext(r.Context())

	for _, m := range media {
		title := m.Stem()
		if m.Title != nil && *m.Title != "" {
			title = *m.Title
		}

		author := "Unknown"
		if m.Artist != nil && *m.Artist != "" {
			author = *m.Artist
		}

		mimeMediaType := "application/octet-stream"
		if m.MediaType != nil {
			mimeMediaType = *m.MediaType
		}

		fmt.Fprintf(w, `
  <entry>
    <title>%s</title>
    <id>%s</id>
    <updated>%s</updated>
    <author><name>%s</name></author>
    <content type="text">%s</content>
    <link rel="http://opds-spec.org/acquisition" href="%s://%s%s/api/raw?path=%s" type="%s"/>
  </entry>`,
			textutil.EscapeXML(title),
			textutil.EscapeXML(m.Path),
			time.Now().Format(time.RFC3339), // Ideally use modification time
			textutil.EscapeXML(author),
			textutil.EscapeXML(m.Path),
			scheme, host, basePath, strings.ReplaceAll(url.QueryEscape(m.Path), "+", "%20"),
			mimeMediaType,
		)
	}

	fmt.Fprint(w, "\n</feed>")
}
