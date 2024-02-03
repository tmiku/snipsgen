package main

import (
	"bytes"
	"database/sql"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

type SnipContent struct {
	InnerHtml    string
	DateHtml     string
	TagsHtml     string
	ContinueHtml string
}

func MakeTagsHtml(tagSlice []string) string {
	htmlOut := ""
	for _, tag := range tagSlice {
		htmlOut = htmlOut + `<span class="tag"> <a href="/snips/tag/` + tag + `.html">` + tag + `</a></span>`
	}
	return htmlOut
}

func MdToHtml(md string) string {

	// create MD parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock | parser.Footnotes
	p := parser.NewWithExtensions(extensions)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank | html.FootnoteNoHRTag
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return string(markdown.ToHTML([]byte(md), p, renderer)) //string(markdown.Render(doc, renderer))
}

func RenderMain(dbPath string) { // this is gross, haven't figured out a smooth refactor yet
	templateHtml := "html/snip.html"
	longTemplateHtml := "html/longsnip.html"
	var snips []string
	homeTemplateBytes, err := os.ReadFile("html/index.html")
	if err != nil {
		panic(err)
	}
	homeTemplate := string(homeTemplateBytes)

	//query to get content
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	rows, err := db.Query(`SELECT snipName, longSnip, snipDate, COALESCE(upperMd,rawMd), rawMd AS md 
		FROM snips WHERE published=TRUE ORDER BY snipDate DESC`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() { //for each snip...
		var snipName string
		var snipDate string
		var homeDisplayMd string
		var rawMd string
		var longSnip bool
		var snipContinue string
		var snipTags []string
		var snipHtml string

		err := rows.Scan(&snipName, &longSnip, &snipDate, &homeDisplayMd, &rawMd)
		if err != nil {
			panic(err)
		}

		// generate the html for tag labels

		tagrows, err := db.Query("SELECT snipTag FROM snipTags WHERE snipName=?", snipName)
		if err != nil {
			panic(err)
		}
		defer tagrows.Close()

		for tagrows.Next() {
			var tag string
			err := tagrows.Scan(&tag)
			if err != nil {
				panic(err)
			}
			snipTags = append(snipTags, tag)
		}
		if longSnip { // create and export dedicated longsnip page if needed

			snipContinue = `
			<p><a class="continue" href="/snips/` + snipName + `.html">Continue reading...</a></p>`

			longContent := SnipContent{InnerHtml: MdToHtml(rawMd),
				DateHtml:     snipDate,
				TagsHtml:     MakeTagsHtml(snipTags),
				ContinueHtml: ""}
			longTmpl, err := template.New(path.Base(longTemplateHtml)).ParseFiles(longTemplateHtml)
			if err != nil {
				panic(err)
			}
			var buf bytes.Buffer
			err = longTmpl.Execute(&buf, longContent)
			if err != nil {
				panic(err)
			}
			err = os.WriteFile("output/"+snipName+".html", buf.Bytes(), 0666)
			if err != nil {
				panic(err)
			}
		} else {
			snipContinue = ""
		}

		// generate the html for this snip
		content := SnipContent{InnerHtml: MdToHtml(homeDisplayMd),
			DateHtml:     snipDate,
			TagsHtml:     MakeTagsHtml(snipTags),
			ContinueHtml: snipContinue}

		tmpl, err := template.New(path.Base(templateHtml)).ParseFiles(templateHtml)
		if err != nil {
			panic(err)
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, content)
		if err != nil {
			panic(err)
		}
		snipHtml = buf.String()

		//append it to the list of all snips
		snips = append(snips, snipHtml)
	}

	//to plug all generated snips into the home template, use a simple string replace instead of go templates
	renderedHome := []byte(strings.Replace(homeTemplate, "bodycontenthere", strings.Join(snips, "\n"), -1))
	err = os.WriteFile("output/index.html", renderedHome, 0666)
	if err != nil {
		panic(err)
	}

}

func RenderTag(dbPath string, tag string) { //this is even grosser.
	templateHtml := "html/snip.html"
	var snips []string
	tagTemplateBytes, err := os.ReadFile("html/tag.html")
	if err != nil {
		panic(err)
	}
	tagTemplate := string(tagTemplateBytes)

	//query to get all snips for the specified tag
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}

	rows, err := db.Query(`
	SELECT snips.snipName, snips.longSnip, snips.snipDate, COALESCE(snips.upperMd,snips.rawMd) AS md 
	FROM snips 
	LEFT OUTER JOIN snipTags ON snips.snipName=snipTags.snipName
	WHERE snips.published=TRUE AND snipTags.snipTag=?
	ORDER BY snips.snipDate DESC`, tag)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() { //for each of those snips...
		var snipName string
		var snipDate string
		var snipMd string
		var longSnip bool
		var snipContinue string
		var snipTags []string
		var snipHtml string

		err := rows.Scan(&snipName, &longSnip, &snipDate, &snipMd)
		if err != nil {
			panic(err)
		}

		// generate tag html
		tagrows, err := db.Query("SELECT snipTag FROM snipTags WHERE snipName=?", snipName)
		if err != nil {
			panic(err)
		}
		defer tagrows.Close()

		for tagrows.Next() {
			var tag string
			err := tagrows.Scan(&tag)
			if err != nil {
				panic(err)
			}
			snipTags = append(snipTags, tag)
		}
		if longSnip { // add Continue Reading link for longsnips (longsnip pages generated in RenderMain())
			snipContinue = `
			<p><a class="continue" href="/snips/` + snipName + `.html">Continue reading...</a></p>`
		} else {
			snipContinue = ""
		}

		//generate the same preview that we put on the home page
		content := SnipContent{InnerHtml: MdToHtml(snipMd),
			DateHtml:     snipDate,
			TagsHtml:     MakeTagsHtml(snipTags),
			ContinueHtml: snipContinue}

		tmpl, err := template.New(path.Base(templateHtml)).ParseFiles(templateHtml)
		if err != nil {
			panic(err)
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, content)
		if err != nil {
			panic(err)
		}
		snipHtml = buf.String()
		//and add it to the list of all snips with this tag
		snips = append(snips, snipHtml)
	}

	//to plug generated snips into the tag template, use a simple string replace instead of go templates
	renderedHome := strings.Replace(tagTemplate, "bodycontenthere", strings.Join(snips, "\n"), -1)
	renderedHome = strings.Replace(renderedHome, "tagheaderhere", `<p class="innerHtml" id="tagheader">Posts with tag: <strong>`+tag+`</strong></p>`, -1)
	// template links assume they're in output/ and not output/tag/. Correct that by string replacing relative links.
	renderedHome = strings.Replace(renderedHome, `"./`, `"../`, -1)
	err = os.WriteFile("output/tag/"+tag+".html", []byte(renderedHome), 0666)
	if err != nil {
		panic(err)
	}
}

func ListAllTags(dbPath string) []string {
	var out []string

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}

	rows, err := db.Query(`SELECT DISTINCT snipTags.snipTag 
		FROM snipTags
		LEFT OUTER JOIN snips ON snipTags.snipName = snips.snipName
		WHERE snips.published=TRUE`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var snipTag string
		err := rows.Scan(&snipTag)
		if err != nil {
			panic(err)
		}
		out = append(out, snipTag)
	}
	return out
}

func RenderAllTags(dbPath string) {
	for _, tag := range ListAllTags(dbPath) {
		RenderTag(dbPath, tag)
	}
}
