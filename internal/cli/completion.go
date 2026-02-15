package cli

import (
	"fmt"
	"strings"
)

func runCompletion(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni completion <bash|zsh|fish>")
	}
	shell := strings.ToLower(strings.TrimSpace(args[0]))
	var script string
	switch shell {
	case "bash":
		script = bashCompletion()
	case "zsh":
		script = zshCompletion()
	case "fish":
		script = fishCompletion()
	default:
		return usageFail(rt, "usage: omni completion <bash|zsh|fish>")
	}
	fmt.Print(script)
	return 0
}

func bashCompletion() string {
	return `# bash completion for omni
_omni_completions() {
  local cur prev words cword
  _init_completion || return

  if [[ $cword -eq 1 ]]; then
    COMPREPLY=( $( compgen -W "setup doctor auth documents models connections folders labels schedules dashboards agentic embed unstable user-attributes admin users scim ai api query jobs completion version help" -- "$cur" ) )
    return
  fi

  case "${words[1]}" in
    auth)
      COMPREPLY=( $( compgen -W "add list remove rm use show whoami" -- "$cur" ) )
      ;;
    query)
      COMPREPLY=( $( compgen -W "run" -- "$cur" ) )
      ;;
    documents)
      COMPREPLY=( $( compgen -W "list get create delete rm rename move draft duplicate favorite access permissions perm label labels queries transfer-ownership" -- "$cur" ) )
      ;;
    models)
      COMPREPLY=( $( compgen -W "list get create refresh validate branch cache-reset topics views fields git migrate content-validator yaml" -- "$cur" ) )
      ;;
    connections)
      COMPREPLY=( $( compgen -W "list create update dbt schedules environments" -- "$cur" ) )
      ;;
    folders)
      COMPREPLY=( $( compgen -W "list create delete rm permissions perm" -- "$cur" ) )
      ;;
    labels)
      COMPREPLY=( $( compgen -W "list get create update delete rm" -- "$cur" ) )
      ;;
    schedules)
      COMPREPLY=( $( compgen -W "list create get update delete rm pause resume trigger recipients transfer-ownership" -- "$cur" ) )
      ;;
    dashboards)
      COMPREPLY=( $( compgen -W "download download-status download-file filters" -- "$cur" ) )
      ;;
    agentic)
      COMPREPLY=( $( compgen -W "submit status cancel result" -- "$cur" ) )
      ;;
    embed)
      COMPREPLY=( $( compgen -W "sso" -- "$cur" ) )
      ;;
    unstable)
      COMPREPLY=( $( compgen -W "documents" -- "$cur" ) )
      ;;
    user-attributes)
      COMPREPLY=( $( compgen -W "list" -- "$cur" ) )
      ;;
    admin)
      COMPREPLY=( $( compgen -W "users groups" -- "$cur" ) )
      ;;
    users)
      COMPREPLY=( $( compgen -W "list-email-only create-email-only create-email-only-bulk roles group-roles" -- "$cur" ) )
      ;;
    scim)
      COMPREPLY=( $( compgen -W "users groups embed-users" -- "$cur" ) )
      ;;
    ai)
      COMPREPLY=( $( compgen -W "generate-query workbook pick-topic" -- "$cur" ) )
      ;;
    api)
      COMPREPLY=( $( compgen -W "call" -- "$cur" ) )
      ;;
    jobs)
      COMPREPLY=( $( compgen -W "status" -- "$cur" ) )
      ;;
    completion)
      COMPREPLY=( $( compgen -W "bash zsh fish" -- "$cur" ) )
      ;;
  esac
}
complete -F _omni_completions omni
`
}

func zshCompletion() string {
	return `#compdef omni

_omni() {
  local -a commands
  commands=(
    'setup:Configure Omni URL + token profile'
    'doctor:Run connectivity and capability checks'
    'auth:Manage profiles and tokens'
    'documents:List and inspect documents'
    'models:List models'
    'connections:Manage connections and environments'
    'folders:List folders'
    'labels:Manage labels'
    'schedules:Manage schedules'
    'dashboards:Manage dashboard downloads and filters'
    'agentic:Run asynchronous AI agentic jobs'
    'embed:Generate embed SSO sessions'
    'unstable:Unstable API routes'
    'user-attributes:List user attribute definitions'
    'admin:Org-key admin commands'
    'users:Org-key user and role management'
    'scim:Org-key SCIM management'
    'ai:Omni AI topic/query generation'
    'api:Raw authenticated Omni API calls'
    'query:Run Omni queries'
    'jobs:Check asynchronous job status'
    'completion:Generate shell completions'
    'version:Print version'
    'help:Show help'
  )

  if (( CURRENT == 2 )); then
    _describe 'command' commands
    return
  fi

  case $words[2] in
    auth)
      _describe 'auth command' 'add list remove rm use show whoami'
      ;;
    query)
      _describe 'query command' 'run'
      ;;
    documents)
      _describe 'documents command' 'list get create delete rm rename move draft duplicate favorite access permissions perm label labels queries transfer-ownership'
      ;;
    models)
      _describe 'models command' 'list get create refresh validate branch cache-reset topics views fields git migrate content-validator yaml'
      ;;
    connections)
      _describe 'connections command' 'list create update dbt schedules environments'
      ;;
    folders)
      _describe 'folders command' 'list create delete rm permissions perm'
      ;;
    labels)
      _describe 'labels command' 'list get create update delete rm'
      ;;
    schedules)
      _describe 'schedules command' 'list create get update delete rm pause resume trigger recipients transfer-ownership'
      ;;
    dashboards)
      _describe 'dashboards command' 'download download-status download-file filters'
      ;;
    agentic)
      _describe 'agentic command' 'submit status cancel result'
      ;;
    embed)
      _describe 'embed command' 'sso'
      ;;
    unstable)
      _describe 'unstable command' 'documents'
      ;;
    user-attributes)
      _describe 'user-attributes command' 'list'
      ;;
    admin)
      _describe 'admin command' 'users groups'
      ;;
    users)
      _describe 'users command' 'list-email-only create-email-only create-email-only-bulk roles group-roles'
      ;;
    scim)
      _describe 'scim command' 'users groups embed-users'
      ;;
    ai)
      _describe 'ai command' 'generate-query workbook pick-topic'
      ;;
    api)
      _describe 'api command' 'call'
      ;;
    jobs)
      _describe 'jobs command' 'status'
      ;;
    completion)
      _describe 'shell' 'bash zsh fish'
      ;;
  esac
}

_omni "$@"
`
}

func fishCompletion() string {
	return `# fish completion for omni
complete -c omni -f -n "__fish_use_subcommand" -a "setup doctor auth documents models connections folders labels schedules dashboards agentic embed unstable user-attributes admin users scim ai api query jobs completion version help"
complete -c omni -f -n "__fish_seen_subcommand_from auth" -a "add list remove rm use show whoami"
complete -c omni -f -n "__fish_seen_subcommand_from documents" -a "list get create delete rm rename move draft duplicate favorite access permissions perm label labels queries transfer-ownership"
complete -c omni -f -n "__fish_seen_subcommand_from models" -a "list get create refresh validate branch cache-reset topics views fields git migrate content-validator yaml"
complete -c omni -f -n "__fish_seen_subcommand_from connections" -a "list create update dbt schedules environments"
complete -c omni -f -n "__fish_seen_subcommand_from folders" -a "list create delete rm permissions perm"
complete -c omni -f -n "__fish_seen_subcommand_from labels" -a "list get create update delete rm"
complete -c omni -f -n "__fish_seen_subcommand_from schedules" -a "list create get update delete rm pause resume trigger recipients transfer-ownership"
complete -c omni -f -n "__fish_seen_subcommand_from dashboards" -a "download download-status download-file filters"
complete -c omni -f -n "__fish_seen_subcommand_from agentic" -a "submit status cancel result"
complete -c omni -f -n "__fish_seen_subcommand_from embed" -a "sso"
complete -c omni -f -n "__fish_seen_subcommand_from unstable" -a "documents"
complete -c omni -f -n "__fish_seen_subcommand_from user-attributes" -a "list"
complete -c omni -f -n "__fish_seen_subcommand_from admin" -a "users groups"
complete -c omni -f -n "__fish_seen_subcommand_from users" -a "list-email-only create-email-only create-email-only-bulk roles group-roles"
complete -c omni -f -n "__fish_seen_subcommand_from scim" -a "users groups embed-users"
complete -c omni -f -n "__fish_seen_subcommand_from ai" -a "generate-query workbook pick-topic"
complete -c omni -f -n "__fish_seen_subcommand_from api" -a "call"
complete -c omni -f -n "__fish_seen_subcommand_from query" -a "run"
complete -c omni -f -n "__fish_seen_subcommand_from jobs" -a "status"
complete -c omni -f -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"
`
}
