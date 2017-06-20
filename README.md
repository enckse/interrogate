survey
===

a basic/primitive survey app for LAN-based survey completion.

## setup

use the epiphyte [repository](https://github.com/epiphyte/repository) to install as a package

```
pacman -S survey
```

which will install to
```
/opt/epiphyte/survey
```

## development

to install necessary dependencies

```
make install
```

### examples

to see the example config files and survey in action

```
make examples
```

### running

by default the make (all) target will run all non-examples from questions in sorted (name) order and write to disk
```
make
```

to alter this behavior you can change OUTPUT to something else
```
make OUTPUT=sqlite
```

and/or provide explicit definitions to execute
```
make OUTPUT=sqlite DEFINITIONS=example
```
