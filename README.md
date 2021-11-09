# func-dynamic-templates

This repo sever as place for experimentation with dynamic templates for func.

## Use cases
As a template developer I want to be able to create project files in a dynamic fashion.
Current implementation only allows static files to be used to create project files.
This is not suitable for instance for Java projects where package structure should be created in accordance with template user wishes.

There are many other possible usages,
for instance if in the future cluster will support some kind of cloudevents discovery
user may be prompted which cloudevents are interesting and the template could generate structure/class for it.

## Idea
For this reason I propose a new mechanism for template creation.
I would like to call it `dynamic templates` or `executable templates` for now.

A dynamic template could be any `executable` file (script binary) satisfying
[executable contract](#executable-contract).
Such an executable would be put into `~/.config/func/repositories`,
then `func` can discover it and expose it to a user in a similar way the static templates are exposed.

## Executable Contract

* The `executable` shall support three sub-commands: `runtimes`, `templates` and `create`.
* The `runtimes` sub-command shall take no parameters.</br>
  The sub-command shall print a JSON array of string to standard output containing a list of supported runtimes.
* The `templates` sub-command shall take exactly one parameter: runtime name. </bf>
  The sub-command shall print a JSON array of string containing a list of supported templates for given runtime.
* The `create` sub-command shall accept exactly one argument: JSON object specified by
[FuncSpec](https://github.com/matejvasek/func-dynamic-templates/blob/main/lib/repository.go#L9) structure.</br>
  The sub-command shall create func project files (including func.yaml) with given FuncSpec.Name and at specified FuncSpec.Root using runtime/template specified by FuncSpec.Runtime/FuncSpec.Template.
* The `executable` may use stdio as it wills. For instance to prompt uses for additional information required for 
  project creation (e.g. group and artifact ids for maven project).
* The `executable` behaviour may be customized by environment variables.

## Go Helper Library

To ease development of binaries satisfying [executable contract](#executable-contract) we will offer `Go` library.

The library is centered around
[Repository](https://github.com/matejvasek/func-dynamic-templates/blob/main/lib/repository.go#L23) and
[Template](https://github.com/matejvasek/func-dynamic-templates/blob/main/lib/repository.go#L16)
interfaces.

Template author will not have to write all the boilerplate related to argument parsing or data serialization.
All that is needed is to implement aforementioned interfaces and then pass it to the
[`NewCommandFromRepository()`](https://github.com/matejvasek/func-dynamic-templates/blob/main/lib/repository.go#L56) function,
and [use it](https://github.com/matejvasek/func-dynamic-templates/blob/main/cmd/quarkus-func-template/main.go#L33) in `main()`.
