# Functions

## gitbase functions

To make some common tasks easier for the user, there are some functions to interact with the aforementioned tables:

|     Name     |                                               Description                                           |
|:-------------|:----------------------------------------------------------------------------------------------------|
|is_remote(reference_name)bool| check if the given reference name is from a remote one                               |
|is_tag(reference_name)bool| check if the given reference name is a tag                                              |
|language(path, [blob])text| gets the language of a file given its path and the optional content of the file         |
|uast(blob, [lang, [xpath]])json_blob| returns an array of UAST nodes as blobs                                       |
|uast_xpath(json_blob, xpath)| performs an XPath query over the given UAST nodes                                     |

## Standard functions

You can check standard functions in [`go-mysql-server` documentation](https://github.com/src-d/go-mysql-server/tree/f813628f2d89f5d6d60e4d847addc89cf6fe7d98#custom-functions).
