---
title: {{printf "%q" .Title}}
date: {{.CreatedAt.Format "2006-01-02T15:04:05Z07:00"}}
draft: false
categories: [{{range $i, $cat := .Categories}}{{if $i}}, {{end}}{{printf "%q" $cat}}{{end}}]
tags: [{{range $i, $tag := .Tags}}{{if $i}}, {{end}}{{printf "%q" $tag}}{{end}}]
author: {{printf "%q" .Author}}
author_title: {{printf "%q" .AuthorTitle}}
deck: {{printf "%q" .Deck}}
source_url: {{printf "%q" .SourceURL}}
source_domain: {{printf "%q" .SourceDomain}}
target_audience: {{printf "%q" .TargetAudience}}
---

{{.Content}}

