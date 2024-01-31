package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

type SnipMetadata struct {
	Name      string
	Date      string
	LongSnip  bool
	Published bool
	Tags      []string
}

type SnipRow struct {
	SnipMetadata
	RawMd   string
	UpperMd string
}

func MetadataFromMD(filepath string) SnipMetadata {
	// reading the whole file in case I want to do any richer preprocessing/metadata.
	mdBytes, err := os.ReadFile(filepath)
	if err != nil {
		panic(err)
	}
	// this is gnarly: the 9 trims of the "[//]: # (" from the markdown comment hack...
	// ... and the string.Index() finds the first newline and cuts it a character before there,
	// catching the ")" from the end of the comment hack.
	// this will be JSON sans linebreaks for the foreseeable future.
	dataLine := mdBytes[9 : strings.Index(string(mdBytes), "\n")-1]

	var metadata SnipMetadata
	err = json.Unmarshal(dataLine, &metadata)
	if err != nil {
		panic(err)
	}
	metadata.Name = filepath[3 : len(filepath)-3] //strip out "md/" and ".md" from filepath

	return metadata
}

func AllMetadata(mdDirPath string) []SnipMetadata {
	var out []SnipMetadata

	mdDir, err := os.Open(mdDirPath)
	if err != nil {
		panic(err)
	}
	mdNames, err := mdDir.Readdirnames(0)
	if err != nil {
		panic(err)
	}

	for _, name := range mdNames {
		out = append(out, MetadataFromMD(mdDirPath+name))
	}

	return out
}

func BuildDb(mdDirPath string, outputPath string) {

	const create string = `
	CREATE TABLE IF NOT EXISTS snips (
		snipName TEXT PRIMARY KEY,
		snipDate TEXT,
		longSnip INTEGER,
		published INTEGER,
		rawMd TEXT,
		upperMd TEXT
	);
	CREATE TABLE IF NOT EXISTS snipTags (
		snipName TEXT,
		snipTag TEXT,
		PRIMARY KEY (snipName, snipTag)
	);`

	// create and open sqlite database
	os.Create(outputPath)
	db, err := sql.Open("sqlite", outputPath)
	if err != nil {
		panic(err)
	}

	// create snips table
	if _, err := db.Exec(create); err != nil {
		panic(err)
	}

	for _, metadata := range AllMetadata(mdDirPath) {
		var rawMd string
		var upperMd string

		//get raw md from filename
		mdBytes, err := os.ReadFile(mdDirPath + metadata.Name + ".md")
		if err != nil {
			panic(err)
		}
		rawMd = string(mdBytes)

		//trim raw md to upper md
		if metadata.LongSnip {
			upperMd = strings.Split(rawMd, `[//]: # (break)`)[0]
		} else {
			upperMd = ""
		}

		// insert snips row for this metadata
		_, err = db.Exec(`
			INSERT INTO snips (snipName, snipDate, longSnip, published, rawMd, upperMd)
			VALUES(?, ?, ?, ?, ?, NULLIF(?,''));`,
			metadata.Name,
			metadata.Date,
			metadata.LongSnip,
			metadata.Published,
			rawMd,
			upperMd)
		if err != nil {
			panic(err)
		}

		// insert snipTags row for this metadata
		for _, tag := range metadata.Tags {
			_, err = db.Exec(`
			INSERT INTO snipTags (snipName, snipTag)
			VALUES(?,?);`,
				metadata.Name,
				tag)
			if err != nil {
				panic(err)
			}
		}

	}
}

func PrintDb(dbPath string) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}

	rows, err := db.Query("SELECT snipName,snipDate,longSnip,published,rawMd,COALESCE(upperMd,'') FROM snips")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var row SnipRow
		err := rows.Scan(&row.Name, &row.Date, &row.LongSnip, &row.Published, &row.RawMd, &row.UpperMd)
		if err != nil {
			panic(err)
		}
		fmt.Println(row)
	}
}
