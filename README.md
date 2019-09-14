Just a proof-of-concept for https://twitter.com/euanc/status/1171778837492973569

It works ... but by loading everything in memory, so really not much different to just fully extracting the contents to disk. Also URLs aren't great filenames & I just swapped the path separators for underscores to sanitise (i.e. no smart creation of subdirectories).

There's a 64 bit linux binary on the [releases](https://github.com/richardlehane/warcmount/releases) page if you want to try it. Otherwise install with golang.

[![asciicast](https://asciinema.org/a/268421.svg)](https://asciinema.org/a/268421)