# Meta Formatter

`metafmt` is an opinionated front-end for various code beautifiers. It is meant to be used from
the command line or integrated into an editor.

It's opinionated, which means that you can't configure it, *this is by design*.


## Supported Formatters

**NOTE**: These have to be installed separately. If one of them isn't installed, `metafmt` will
skip the file and do nothing.

* C/C++: [clang-format](http://clang.llvm.org/docs/ClangFormat.html);
* CSS: [cssbeautify]();
* Go: [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports);
* JavaScript: [semistandard-format](https://github.com/ricardofbarros/semistandard-format);
* JSON: [jsonlint](https://github.com/zaach/jsonlint);
* Python:
  - [autopep8](https://github.com/hhatto/autopep8);
  - [isort](https://github.com/timothycrosley/isort);
* SASS/SCSS: [ruby-sass](http://sass-lang.com/install);
