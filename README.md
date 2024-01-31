# snipsgen

Snipsgen is a specialized static site generator for the blog on my personal website.

After running the generator, take the following steps to integrate it with the main website:
- Copy the output directory into the Snips directory in the main site.
- Add any missing images as described in the output of the program.

**NOTE:* Be careful with the output folder - ideally, you should use the clearOutput script to remove just the .html files. Otherwise you'll enter a world of pain and need to manually re-obtain those images and rewrite the CSS file.