Hakmes (it means "cleaver" in Dutch) is a REST-based
content-addressed storage service intended for large files.

In a nutshell, you `POST` a file to it and it gives you back a hash
(currently just a SHA1 of the contents of the file). Later, you can
make a `GET` request with that hash as the key and it will return the
contents of the file. It is optimized to work with large (multi-GB at
least) files.

Hakmes functions as a front-end to a
[Cask](https://github.com/thraxil/cask/) cluster, which handles
efficiently storing data replicated across multiple nodes. Hakmes
takes the file that you upload, splits it into a number of chunks, and
stores those chunks to the Cask cluster. When you retrieve the file,
it pulls the chunks down and reassembles the file on the fly.
