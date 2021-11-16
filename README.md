# func-dynamic-templates

This repo sever as place for experimentation with dynamic templates for func.

## Use cases
As a template developer I want to be able to create project files in a dynamic fashion.
Current implementation only allows static files to be used to create project files.
This is not suitable for instance for Java projects where package structure should be created in accordance with template user wishes.

There are many other possible usages,
for instance if in the future cluster will support some kind of cloudevents discovery
user may be prompted which cloudevents are interesting and the template could generate structure/class for it.

Here are three common use-cases for dynamic code generation:
* Automatic code generation for structures (CloudEvents types / payloads)
* Library path generation for e.g. Java includes.
* Optional feature inclusion for functionality which may need configuration (trace/metrics prefix, for example)


## Idea
For this reason I propose a new mechanism for template creation.
I would like to call it `dynamic templates` or `executable templates` for now.

A dynamic template could be any `executable` file (script binary) satisfying
[executable contract](#executable-contract).
Such an executable would be put into `~/.config/func/repositories`,
then `func` can discover it and expose it to a user in a similar way the static templates are exposed.

Other possibility would be to `containerize` the executable.
Then the image name would be specified in a configuration file under `~/.config/func`.

## Executable Contract A

* The `executable` shall support three sub-commands: `runtimes`, `templates` and `create`.
* The `runtimes` sub-command shall take no parameters.</br>
  The sub-command shall print a JSON array of string to standard output containing a list of supported runtimes.
* The `templates` sub-command shall take exactly one parameter: runtime name. </br>
  The sub-command shall print a JSON array of string containing a list of supported templates for given runtime.
* The `create` sub-command shall accept exactly one argument: JSON object specified by
[FuncSpec](https://github.com/matejvasek/func-dynamic-templates/blob/main/lib/repository.go#L9) structure.</br>
  The sub-command shall create func project files (including func.yaml) with given FuncSpec.Name and at specified FuncSpec.Root using runtime/template specified by FuncSpec.Runtime/FuncSpec.Template.
* The `executable` may use stdio as it wills. For instance to prompt uses for additional information required for 
  project creation (e.g. group and artifact ids for maven project).
* The `executable` behaviour may be customized by environment variables.

The advantage of the `Contract A` is simpler implementation,
but it would be usable mostly just for `func`.

## Executable Contract B

* The `executable` shall support three sub-commands: `runtimes`, `templates` and `create`.
* The `runtimes` sub-command shall take no parameters.</br>
  The sub-command shall print a JSON array of string to standard output containing a list of supported runtimes.
* The `templates` sub-command shall take exactly one parameter: runtime name. </br>
  The sub-command shall print a JSON Object where keys are names of templates and values are specification
  of questions/parameters needed by a template.</br>
  Example of such a JSON: 
  ```json
  {
    "my-java-template": {
      "questions": [
        {
          "name": "artifact-id",
          "type": "string"
        },
        {
          "name": "group-id",
          "type": "string"
        },
        {
          "name": "build-system",
          "type": "select",
          "items": [
            "maven",
            "gradle"
          ]
        },
        {
          "name": "hypothetical",
          "type": "string",
          "depends-on": "build-system",
          "ask-when": {
            "condition": {
              "type": "equals",
              "rhs": ".build-system",
              "lhs": "'maven'"
            }
          }
        }
      ]
    }
  }
  ```
* The `create` sub-command shall accept four argument:
  * --name: name of the Function project
  * --language: language of the Function project
  * --template: the template to create the Function project</br>
  The sub-command shall use `stdio` in a specific way:</br></br>
  On `stdin` there would be additional parameters provided as an JSON Object based on specification provided
  by the `templates` sub-command for given language/template.</br>
  Example of such a JSON:
  ```json
  {
    "answers": [
      {
        "name": "artifact-id",
        "value": "my-art"
      },
      {
        "name": "group-id",
        "value": "my.grp"
      },
      {
        "name": "build-system",
        "value": "gradle"
      }
    ]
  }
  ```
  The sub-command is supposed to return project files as `tar` archive on `stdout`.

The advantage of the `Contract B` is that it could be better for external tools,
but implementation would be more difficult.

## Containerization

As mentioned before the `excutable` could be containerized.

Advantages would be:
* Easier distribution. Multi-arch images would make it easier to support multiple platforms.
Without it users would have to download the righ binaries for themselves.
* Better isolation. If the `executable` is not trusted it's safer to limit its access
to filesystem, network and environment variables.</br>
On the other hand this would limit `executable` no only in doing bad things but also good things.
For instance template could ask `k8s` cluster about available event sources and ask users if they
want to use some as a source for their function and generate code accordingly.

## Go Helper Library

To ease development of binaries satisfying `executable contract` we will offer `Go` library.

The library is centered around
`Repository`and `Template` interfaces.

<b>Contract A</b>
```go
// For contract A
type Template interface {
	Name() string
	Runtime() string
	// Write creates all project files including func.yaml
	Write(ctx context.Context, projectName, destDir string) error
}

type Repository interface {
	Runtimes(ctx context.Context) (runtimes []string, err error)
	Templates(ctx context.Context, runtime string) (templates []string, err error)
	Template(ctx context.Context, runtime, templateName string) (Template, error)
}
```

<b>Contract B</b>
```go
// For contract B
type Question interface {
	// TODO
}

type Template interface {
    Name() string
    Runtime() string
    Questions(ctx context.Context) []Question
    // Returns tar stream of project files including func.yaml
    Write(ctx context.Context, projectName string, answers map[string]string) (io.Reader, error)
}

type Repository interface {
    Runtimes(ctx context.Context) (runtimes []string, err error)
    Templates(ctx context.Context, runtime string) (templates []string, err error)
    Template(ctx context.Context, runtime, templateName string) (Template, error)
}
```

Template author will not have to write all the boilerplate related to argument parsing or data serialization.
All that is needed is to implement aforementioned interfaces and then pass it to the
[`NewCommandFromRepository()`](https://github.com/matejvasek/func-dynamic-templates/blob/main/lib/repository.go#L56) function,
and [use it](https://github.com/matejvasek/func-dynamic-templates/blob/main/cmd/quarkus-func-template/main.go#L33) in `main()`.
