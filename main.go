package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
)

func main() {
	BuildDb("md/", "snips.db")
	RenderMain("snips.db")
	RenderAllTags("snips.db")
	validateImages("./output")
}

func validateImages(outputDir string) {
	imagesLinked := mapset.NewSet[string]()
	imagesSaved := mapset.NewSet[string]()
	imageDir := outputDir + "/images"

	dir, err := os.Open(outputDir)
	if err != nil {
		panic(err)
	}
	names, err := dir.Readdirnames(0)
	if err != nil {
		panic(err)
	}

	imgRegEx := regexp.MustCompile(`<img ?.* src="\./images/([^"]*)"`)

	for _, name := range names {
		if strings.HasSuffix(name, ".html") {
			fmt.Println("reading file: " + name)
			snipBytes, err := os.ReadFile(outputDir + "/" + name)
			if err != nil {
				panic(err)
			}
			for _, match := range imgRegEx.FindAllSubmatch(snipBytes, -1) {
				imagesLinked.Add(string(match[1]))
			}

		}
	}

	//get list of images

	dir, err = os.Open(imageDir)
	if err != nil {
		panic(err)
	}
	names, err = dir.Readdirnames(0)
	if err != nil {
		panic(err)
	}

	for _, name := range names {
		imagesSaved.Add(name)
	}

	notSaved := imagesLinked.Difference(imagesSaved)
	notLinked := imagesSaved.Difference(imagesLinked)

	if notSaved.Cardinality() > 0 {
		fmt.Println("The following images are referenced but not in the images folder:", notSaved)
	} else if notLinked.Cardinality() > 0 {
		fmt.Println("The following images are in the images folder but not used:", notLinked)
	} else {
		fmt.Println("Linked and saved images match!")
	}
}
