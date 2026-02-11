package main

import (
	"flag"
	"fmt"
	"os"
)

func completionCmd() {
	fs := flag.NewFlagSet("completion", flag.ExitOnError)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gottp completion <bash|zsh|fish>\n\n")
		fmt.Fprintf(os.Stderr, "Generate shell completion scripts.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  # Bash\n")
		fmt.Fprintf(os.Stderr, "  gottp completion bash > /usr/local/etc/bash_completion.d/gottp\n")
		fmt.Fprintf(os.Stderr, "  # Zsh\n")
		fmt.Fprintf(os.Stderr, "  gottp completion zsh > \"${fpath[1]}/_gottp\"\n")
		fmt.Fprintf(os.Stderr, "  # Fish\n")
		fmt.Fprintf(os.Stderr, "  gottp completion fish > ~/.config/fish/completions/gottp.fish\n")
	}

	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: shell name is required (bash, zsh, or fish)\n\n")
		fs.Usage()
		os.Exit(1)
	}

	shell := fs.Arg(0)
	switch shell {
	case "bash":
		fmt.Print(generateBashCompletion())
	case "zsh":
		fmt.Print(generateZshCompletion())
	case "fish":
		fmt.Print(generateFishCompletion())
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported shell %q (use bash, zsh, or fish)\n", shell)
		os.Exit(1)
	}
}

func generateBashCompletion() string {
	return `# bash completion for gottp                              -*- shell-script -*-

_gottp() {
    local cur prev words cword
    _init_completion || return

    local commands="run init validate fmt import export mock completion version help"

    # Flags per subcommand
    local run_flags="--env --request --folder --workflow --output --verbose --timeout --perf-save --perf-baseline --perf-threshold"
    local init_flags="--name --output --with-env"
    local validate_flags=""
    local fmt_flags="-w --check"
    local import_flags="--format --output"
    local export_flags="--format --request --output"
    local mock_flags=""
    local completion_flags=""

    # Output format values
    local output_formats="text json junit"
    local export_formats="curl har postman insomnia"
    local import_formats="curl postman insomnia openapi har"
    local shells="bash zsh fish"

    if [[ ${cword} -eq 1 ]]; then
        COMPREPLY=($(compgen -W "${commands}" -- "${cur}"))
        return
    fi

    local command="${words[1]}"

    # Complete flag values
    case "${prev}" in
        --output)
            case "${command}" in
                run)
                    COMPREPLY=($(compgen -W "${output_formats}" -- "${cur}"))
                    return
                    ;;
                *)
                    # File completion
                    _filedir
                    return
                    ;;
            esac
            ;;
        --format)
            case "${command}" in
                export)
                    COMPREPLY=($(compgen -W "${export_formats}" -- "${cur}"))
                    return
                    ;;
                import)
                    COMPREPLY=($(compgen -W "${import_formats}" -- "${cur}"))
                    return
                    ;;
            esac
            ;;
        --env|--request|--folder|--workflow|--name|--timeout|--perf-threshold)
            # These take user-provided values, no completion
            return
            ;;
        --perf-save|--perf-baseline)
            # File completion for baseline files
            _filedir
            return
            ;;
    esac

    # Complete flags for each subcommand
    case "${command}" in
        run)
            if [[ "${cur}" == -* ]]; then
                COMPREPLY=($(compgen -W "${run_flags}" -- "${cur}"))
            else
                # Complete .gottp.yaml files
                COMPREPLY=($(compgen -f -X '!*.gottp.yaml' -- "${cur}"))
                _filedir -d
            fi
            ;;
        init)
            if [[ "${cur}" == -* ]]; then
                COMPREPLY=($(compgen -W "${init_flags}" -- "${cur}"))
            fi
            ;;
        validate)
            COMPREPLY=($(compgen -f -X '!*.gottp.yaml' -- "${cur}"))
            _filedir -d
            ;;
        fmt)
            if [[ "${cur}" == -* ]]; then
                COMPREPLY=($(compgen -W "${fmt_flags}" -- "${cur}"))
            else
                COMPREPLY=($(compgen -f -X '!*.gottp.yaml' -- "${cur}"))
                _filedir -d
            fi
            ;;
        import)
            if [[ "${cur}" == -* ]]; then
                COMPREPLY=($(compgen -W "${import_flags}" -- "${cur}"))
            else
                _filedir
            fi
            ;;
        export)
            if [[ "${cur}" == -* ]]; then
                COMPREPLY=($(compgen -W "${export_flags}" -- "${cur}"))
            else
                COMPREPLY=($(compgen -f -X '!*.gottp.yaml' -- "${cur}"))
                _filedir -d
            fi
            ;;
        completion)
            COMPREPLY=($(compgen -W "${shells}" -- "${cur}"))
            ;;
    esac
}

complete -F _gottp gottp
`
}

func generateZshCompletion() string {
	return `#compdef gottp

# zsh completion for gottp

_gottp() {
    local -a commands
    commands=(
        'run:Run API requests headlessly from a collection file'
        'init:Create a new .gottp.yaml collection interactively'
        'validate:Validate collection and environment YAML files'
        'fmt:Format and normalize collection YAML files'
        'import:Import collection from cURL/Postman/Insomnia/OpenAPI/HAR'
        'export:Export collection to cURL/HAR/Postman/Insomnia format'
        'mock:Start a mock server from a collection'
        'completion:Generate shell completion scripts'
        'version:Print version information'
        'help:Show help message'
    )

    _arguments -C \
        '1:command:->command' \
        '*::arg:->args'

    case $state in
        command)
            _describe -t commands 'gottp commands' commands
            ;;
        args)
            case $words[1] in
                run)
                    _arguments \
                        '--env[Environment name to use]:environment name:' \
                        '--request[Run a single request by name]:request name:' \
                        '--folder[Run all requests in a folder]:folder name:' \
                        '--workflow[Run a named workflow]:workflow name:' \
                        '--output[Output format]:format:(text json junit)' \
                        '--verbose[Show response bodies and headers]' \
                        '--timeout[Request timeout]:timeout:' \
                        '--perf-save[Save timing results as a performance baseline file]:file:_files' \
                        '--perf-baseline[Compare timings against a baseline file]:file:_files' \
                        '--perf-threshold[Regression threshold percentage]:threshold:' \
                        '*:collection file:_files -g "*.gottp.yaml"'
                    ;;
                init)
                    _arguments \
                        '--name[Collection name]:name:' \
                        '--output[Output file path]:output file:_files -g "*.gottp.yaml"' \
                        '--with-env[Also create an environments.yaml file]'
                    ;;
                validate)
                    _arguments \
                        '*:collection file:_files -g "*.gottp.yaml"'
                    ;;
                fmt)
                    _arguments \
                        '-w[Write result to file instead of stdout]' \
                        '--check[Check if files are formatted]' \
                        '*:collection file:_files -g "*.gottp.yaml"'
                    ;;
                import)
                    _arguments \
                        '--format[Force format]:format:(curl postman insomnia openapi har)' \
                        '--output[Output .gottp.yaml file path]:output file:_files -g "*.gottp.yaml"' \
                        '*:input file:_files'
                    ;;
                export)
                    _arguments \
                        '--format[Export format]:format:(curl har postman insomnia)' \
                        '--request[Export a single request by name]:request name:' \
                        '--output[Output file path]:output file:_files' \
                        '*:collection file:_files -g "*.gottp.yaml"'
                    ;;
                completion)
                    _arguments \
                        '1:shell:(bash zsh fish)'
                    ;;
            esac
            ;;
    esac
}

_gottp "$@"
`
}

func generateFishCompletion() string {
	return `# fish completion for gottp

# Disable file completions by default
complete -c gottp -f

# Subcommands
complete -c gottp -n '__fish_use_subcommand' -a run -d 'Run API requests headlessly from a collection file'
complete -c gottp -n '__fish_use_subcommand' -a init -d 'Create a new .gottp.yaml collection interactively'
complete -c gottp -n '__fish_use_subcommand' -a validate -d 'Validate collection and environment YAML files'
complete -c gottp -n '__fish_use_subcommand' -a fmt -d 'Format and normalize collection YAML files'
complete -c gottp -n '__fish_use_subcommand' -a import -d 'Import collection from cURL/Postman/Insomnia/OpenAPI/HAR'
complete -c gottp -n '__fish_use_subcommand' -a export -d 'Export collection to cURL/HAR/Postman/Insomnia format'
complete -c gottp -n '__fish_use_subcommand' -a mock -d 'Start a mock server from a collection'
complete -c gottp -n '__fish_use_subcommand' -a completion -d 'Generate shell completion scripts'
complete -c gottp -n '__fish_use_subcommand' -a version -d 'Print version information'
complete -c gottp -n '__fish_use_subcommand' -a help -d 'Show help message'

# run flags
complete -c gottp -n '__fish_seen_subcommand_from run' -l env -d 'Environment name to use' -r
complete -c gottp -n '__fish_seen_subcommand_from run' -l request -d 'Run a single request by name' -r
complete -c gottp -n '__fish_seen_subcommand_from run' -l folder -d 'Run all requests in a folder' -r
complete -c gottp -n '__fish_seen_subcommand_from run' -l workflow -d 'Run a named workflow' -r
complete -c gottp -n '__fish_seen_subcommand_from run' -l output -d 'Output format' -ra 'text json junit'
complete -c gottp -n '__fish_seen_subcommand_from run' -l verbose -d 'Show response bodies and headers'
complete -c gottp -n '__fish_seen_subcommand_from run' -l timeout -d 'Request timeout' -r
complete -c gottp -n '__fish_seen_subcommand_from run' -l perf-save -d 'Save timing results as a performance baseline file' -rF
complete -c gottp -n '__fish_seen_subcommand_from run' -l perf-baseline -d 'Compare timings against a baseline file' -rF
complete -c gottp -n '__fish_seen_subcommand_from run' -l perf-threshold -d 'Regression threshold percentage' -r
complete -c gottp -n '__fish_seen_subcommand_from run' -F

# init flags
complete -c gottp -n '__fish_seen_subcommand_from init' -l name -d 'Collection name' -r
complete -c gottp -n '__fish_seen_subcommand_from init' -l output -d 'Output file path' -rF
complete -c gottp -n '__fish_seen_subcommand_from init' -l with-env -d 'Also create an environments.yaml file'

# validate - file completion
complete -c gottp -n '__fish_seen_subcommand_from validate' -F

# fmt flags
complete -c gottp -n '__fish_seen_subcommand_from fmt' -s w -d 'Write result to file instead of stdout'
complete -c gottp -n '__fish_seen_subcommand_from fmt' -l check -d 'Check if files are formatted'
complete -c gottp -n '__fish_seen_subcommand_from fmt' -F

# import flags
complete -c gottp -n '__fish_seen_subcommand_from import' -l format -d 'Force format' -ra 'curl postman insomnia openapi har'
complete -c gottp -n '__fish_seen_subcommand_from import' -l output -d 'Output .gottp.yaml file path' -rF
complete -c gottp -n '__fish_seen_subcommand_from import' -F

# export flags
complete -c gottp -n '__fish_seen_subcommand_from export' -l format -d 'Export format' -ra 'curl har postman insomnia'
complete -c gottp -n '__fish_seen_subcommand_from export' -l request -d 'Export a single request by name' -r
complete -c gottp -n '__fish_seen_subcommand_from export' -l output -d 'Output file path' -rF
complete -c gottp -n '__fish_seen_subcommand_from export' -F

# completion - shell names
complete -c gottp -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish' -d 'Shell type'
`
}
