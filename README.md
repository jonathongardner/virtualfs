# VirtualFS
This is for creating virtual filesystems. Its to make extracting nested archives easier.
For example the following could be built:
```
foo.tar
|  README.md
|  foo.go
|---- bar
|    |  bar.go
|    |  another_bar.go
|    |---- compressed.tar.gz
|         | something1.txt
|         | something2.txt
|
|---- baz.tar
     |  ANOTHER_README.md
     |  baz.go
     |---- another_bar_folder
          | bar.go
          | another_bar.go

```

# Features
- Stores the same file (base on checksums) once

# TODO
- Handle orphaned shas
- Move io stuff to fifo
- Cleanup test to be more goie
- Write files as gzip and return reader/closer/reseter
- iterate over each unique file and get path
- Add limited cached writer/reader and write that if its in the limit