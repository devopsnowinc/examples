# Creating an Installer Bundle

## Prereq

On your machine, make sure you have [Makeself](https://makeself.io) installed

e.g.,
```
$ sudo apt install makeself
```

### Creating the bundle

Makeself will bundle the target directory and run the startup/setup script you specify against it when a user runs the resulting bundle.

So, in this case, if you are in the current (`./`) directory, you'd run the following to create the executable bundle:

```
$ makeself ./ installer.sh "OpsVerse Agent Installer" ./setup.sh
```

This will bundle everything in `./` and create `installer.sh` which will unbundle and execute "./setup.sh" when a user runs `installer.sh` (in this case, the executable bundle installer.sh has label "OpsVerse Agent Installer" - just a name it spits out when running)
