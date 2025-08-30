---
title: "{{.Title}}"
date: {{.CreatedAt.Format "2006-01-02T15:04:05Z07:00"}}
draft: false
categories: [{{range $i, $cat := .Categories}}{{if $i}}, {{end}}"{{$cat}}"{{end}}]
tags: [{{range $i, $tag := .Tags}}{{if $i}}, {{end}}"{{$tag}}"{{end}}]
author: "{{.Author}}"
author_title: "{{.AuthorTitle}}"
deck: "{{.Deck}}"
source_url: "{{.SourceURL}}"
source_domain: "{{.SourceDomain}}"
---

{{.Content}}

