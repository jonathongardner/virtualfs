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
- To Json back to virtual
- Symlinks