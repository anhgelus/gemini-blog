# Gemini blog

This is the software behind my [Gemini](https://geminiprotocol.net/) blog.

Gemini is a simple hypertext protocol like Gopher.
Its goal is to build an internet space focused on content and not on distractions like JS, CSS or any other things.
It uses a custom syntax which looks like the markdown and cannot be modified.
Gemini is a *low tech*.

My blog handles static files (in the folder `public`), and use SSR to generate all content inside `/films/` (and 
`/film`).
