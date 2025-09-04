# Welcome to Your Wiki

This is your personal wiki! You can write documentation in markdown format.

## Features

- **Wiki Links**: Use [[Page Name]] to link to other pages
- **Footnotes**: Use [^1] to create footnotes[^1]
- **Standard Markdown**: All the usual markdown features work
- **Persistent Storage**: All pages are saved to SQLite database
- **Portable**: Easy to move between machines with Docker

## Example Wiki Link

Try creating a new page and linking to it: [[Getting Started]]

## Example Footnotes

This text has a footnote[^2].

And here's another reference[^1] to the first footnote.

[^1]: This is the first footnote. It will appear at the bottom of the page.
[^2]: This is the second footnote with some more detailed information.

## Quick Markdown Reference

- Use # for headers
- Use **bold** for bold text
- Use *italic* for italic text
- Use - for bullet points
- Use tildes (~~~) or indent with four spaces for code blocks

## API Endpoints

This wiki runs on a Go backend with the following endpoints:

- GET /api/pages - List all pages
- GET /api/pages/{title} - Get specific page
- POST /api/pages - Create new page
- PUT /api/pages/{title} - Update existing page
- DELETE /api/pages/{title} - Delete page
