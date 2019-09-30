# HuJSON - Human JSON

The HuJSON decoder is a JSON decoder that also allows

- comments, both `/* ... */` and `// to end of line`
- trailing commas on arrays and object members

It is a soft fork of the Go standard library `encoding/json` package.
The plan is to merge in all changes from each Go release.

Currently HuJSON is based on Go 1.13.

## Grammar

The changes to the [JSON grammar](https://json.org) are:

```
--- grammar.json
+++ grammar.hujson
@@ -1,13 +1,31 @@
 members
 	member
+	member ',' ws
 	member ',' members
 
 elements
 	element
+	element ',' ws
 	element ',' elements
 
+comments
+	"*/"
+	comment comments
+
+comment
+	'0000' . '10FFFF'
+
+linecomments
+	'\n'
+	linecomment
+
+linecomment
+	'0000' . '10FFFF' - '\n'
+
 ws
 	""
+	"/*" comments
+	"//" linecomments
 	'0020' ws
 	'000A' ws
 	'000D' ws
```
