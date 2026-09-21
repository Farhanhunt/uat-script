#!/usr/bin/env bash
# =============================================================================
#  Mandatory Smoke Test Patterns (Improved Version)
#  Sourced from resource/mandatory-tests.json
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
#  Caching & Internal State
# ---------------------------------------------------------------------------
declare -g _MANDATORY_JSON_CACHE=""
declare -g -A _MANDATORY_LANG_MAP=(
    [go]="go"
    [golang]="go"
    [python]="python"
    [py]="python"
    [php]="php"
    [node]="node"
    [nodejs]="node"
    [javascript]="node"
    [java]="java"
)

# ---------------------------------------------------------------------------
#  Color helpers (auto-disabled when not in terminal)
# ---------------------------------------------------------------------------
_mandatory_color() {
    if [ -t 1 ]; then
        case "$1" in
            red)    printf '\033[0;31m' ;;
            green)  printf '\033[0;32m' ;;
            yellow) printf '\033[1;33m' ;;
            blue)   printf '\033[0;34m' ;;
            reset)  printf '\033[0m' ;;
        esac
    fi
}

_mandatory_error() {
    echo "$(_mandatory_color red)ERROR:$(_mandatory_color reset) $*" >&2
}

_mandatory_warn() {
    echo "$(_mandatory_color yellow)WARNING:$(_mandatory_color reset) $*" >&2
}

# ---------------------------------------------------------------------------
#  Path Resolution
# ---------------------------------------------------------------------------
_mandatory_runners_dir() {
    # Check language-specific runner dirs
    local lang_dirs=(
        "GO_RUNNERS_DIR"
        "NODE_RUNNERS_DIR"
        "PYTHON_RUNNERS_DIR"
        "PHP_RUNNERS_DIR"
        "JAVA_RUNNERS_DIR"
    )
    local dir_var
    for dir_var in "${lang_dirs[@]}"; do
        if [ -n "${!dir_var:-}" ]; then
            echo "${!dir_var}"
            return 0
        fi
    done

    # Fallback: directory of this script (for direct bash sourcing)
    if [ -n "${BASH_SOURCE[0]:-}" ]; then
        local script_dir
        script_dir=$(CDPATH= cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)
        echo "$script_dir"
        return 0
    fi

    _mandatory_error "Set GO_RUNNERS_DIR (or any *_RUNNERS_DIR) before sourcing mandatory-tests.sh"
    return 1
}

_mandatory_tests_json_path() {
    # 1. Explicit override via env var
    if [ -n "${MANDATORY_TESTS_JSON:-}" ]; then
        if [ -f "$MANDATORY_TESTS_JSON" ]; then
            echo "$MANDATORY_TESTS_JSON"
            return 0
        else
            _mandatory_error "MANDATORY_TESTS_JSON is set but file not found: $MANDATORY_TESTS_JSON"
            return 1
        fi
    fi

    # 2. Default location relative to runners dir
    local runners_dir project_root candidate
    runners_dir=$(_mandatory_runners_dir) || return 1
    project_root=$(dirname "$runners_dir")
    candidate="$project_root/resource/mandatory-tests.json"

    if [ -f "$candidate" ]; then
        echo "$candidate"
        return 0
    fi

    _mandatory_error "mandatory-tests.json not found at: $candidate"
    _mandatory_error "Hint: Set MANDATORY_TESTS_JSON environment variable to override location"
    return 1
}

# ---------------------------------------------------------------------------
#  Cached JSON access (read file only ONCE per shell session)
# ---------------------------------------------------------------------------
_mandatory_load_json() {
    if [ -z "$_MANDATORY_JSON_CACHE" ]; then
        local json_path
        json_path=$(_mandatory_tests_json_path) || return 1
        _MANDATORY_JSON_CACHE=$(cat "$json_path")
    fi
    echo "$_MANDATORY_JSON_CACHE"
}

_mandatory_jq() {
    local json
    json=$(_mandatory_load_json) || return 1
    jq "$@" <<< "$json"
}

# ---------------------------------------------------------------------------
#  Name / Key Conversion
# ---------------------------------------------------------------------------
_mandatory_product_key() {
    echo "$1"
}

_mandatory_java_module_to_key() {
    case "$1" in
        paymentgateway) echo "payment_gateway" ;;
        widget)         echo "widget" ;;
        disbursement)   echo "disbursement" ;;
        *)              echo "$1" ;;
    esac
}

_mandatory_normalize_lang() {
    local input="${1:-}"
    local lower
    lower=$(echo "$input" | tr '[:upper:]' '[:lower:]')
    echo "${_MANDATORY_LANG_MAP[$lower]:-$lower}"
}

# ---------------------------------------------------------------------------
#  Schedule Helpers
# ---------------------------------------------------------------------------
_mandatory_uptime_schedule() {
    [ "${MANDATORY_UPTIME_SCHEDULE:-false}" = "true" ] \
        || [ "${PIPELINE_TRIGGER_SOURCE:-}" = "go-mandatory-schedule" ]
}

_mandatory_go_uptime_jq_filter() {
    echo 'map(select(
        . != "TestTransactionSuccessNotify"
        and . != "TestInternalServerErrorNotify"
        and . != "TestExpiredNotify"
    ))'
}

# ---------------------------------------------------------------------------
#  Validation Helpers
# ---------------------------------------------------------------------------
_mandatory_validate_product() {
    local product="$1"
    local exists
    exists=$(_mandatory_jq -r --arg p "$product" '(.products | has($p))')
    if [ "$exists" != "true" ]; then
        _mandatory_error "Product '$product' not found in mandatory-tests.json"
        _mandatory_warn "Available products: $(mandatory_list_products | tr '\n' ' ')"
        return 1
    fi
    return 0
}

_mandatory_validate_language() {
    local product="$1" lang="$2"
    local exists
    exists=$(_mandatory_jq -r --arg p "$product" --arg l "$lang" '(.products[$p] | has($l))')
    if [ "$exists" != "true" ]; then
        _mandatory_error "Language '$lang' not defined for product '$product'"
        _mandatory_warn "Available languages for '$product': $(mandatory_list_languages "$product" | tr '\n' ' ')"
        return 1
    fi
    return 0
}

# ---------------------------------------------------------------------------
#  Core: Generic Pattern Generator (used by all language-specific functions)
# ---------------------------------------------------------------------------
_mandatory_generic_pattern() {
    local product="$1" lang="$2"

    # Validate inputs
    _mandatory_validate_product "$product" || return 1
    _mandatory_validate_language "$product" "$lang" || return 1

    # Special handling: Go uptime schedule excludes notify tests
    if [ "$lang" = "go" ] && _mandatory_uptime_schedule; then
        local filter
        filter=$(_mandatory_go_uptime_jq_filter)
        _mandatory_jq -r --arg p "$product" --arg l "$lang" \
            ".products[\$p][\$l] | $filter | join(\"|\")"
    else
        _mandatory_jq -r --arg p "$product" --arg l "$lang" \
            '.products[$p][$l] | join("|")'
    fi
}

_mandatory_generic_count() {
    local product="$1" lang="$2"

    _mandatory_validate_product "$product" || return 1
    _mandatory_validate_language "$product" "$lang" || return 1

    if [ "$lang" = "go" ] && _mandatory_uptime_schedule; then
        local filter
        filter=$(_mandatory_go_uptime_jq_filter)
        _mandatory_jq -r --arg p "$product" --arg l "$lang" \
            ".products[\$p][\$l] | $filter | length"
    else
        _mandatory_jq -r --arg p "$product" --arg l "$lang" \
            '.products[$p][$l] | length'
    fi
}

# ---------------------------------------------------------------------------
#  Public API — Language-Specific Pattern Functions (BACKWARD COMPATIBLE)
# ---------------------------------------------------------------------------
mandatory_go_pattern() {
    local module
    module=$(_mandatory_product_key "$1")
    _mandatory_generic_pattern "$module" "go"
}

mandatory_python_pattern() {
    local module
    module=$(_mandatory_product_key "$1")
    _mandatory_generic_pattern "$module" "python"
}

mandatory_php_pattern() {
    local module
    module=$(_mandatory_product_key "$1")
    _mandatory_generic_pattern "$module" "php"
}

mandatory_node_pattern() {
    local module
    module=$(_mandatory_product_key "$1")
    _mandatory_generic_pattern "$module" "node"
}

mandatory_java_pattern() {
    local key
    key=$(_mandatory_java_module_to_key "$1")
    _mandatory_validate_product "$key" || return 1
    _mandatory_validate_language "$key" "java" || return 1
    _mandatory_jq -r --arg p "$key" '
        .products[$p].java
        | map(.class + "#" + (.methods | join("+")))
        | join(",")
    '
}

# ---------------------------------------------------------------------------
#  Public API — Count Functions
# ---------------------------------------------------------------------------
mandatory_go_count() {
    local module
    module=$(_mandatory_product_key "$1")
    _mandatory_generic_count "$module" "go"
}

mandatory_count() {
    local product lang
    product=$(_mandatory_product_key "$1")
    lang=$(_mandatory_normalize_lang "${2:-go}")
    _mandatory_generic_count "$product" "$lang"
}

# ---------------------------------------------------------------------------
#  Public API — Aliases (BACKWARD COMPATIBLE)
# ---------------------------------------------------------------------------
get_mandatory_pattern_for_module() {
    mandatory_go_pattern "$1"
}

get_mandatory_pattern_for_folder() {
    mandatory_go_pattern "$1"
}

# ---------------------------------------------------------------------------
#  🆕 NEW: Discovery & Helper Functions
# ---------------------------------------------------------------------------

# List all available products in the JSON
mandatory_list_products() {
    _mandatory_jq -r '.products | keys[]'
}

# List all languages available for a product
mandatory_list_languages() {
    local product
    product=$(_mandatory_product_key "$1")
    _mandatory_validate_product "$product" || return 1
    _mandatory_jq -r --arg p "$product" '.products[$p] | keys[]'
}

# Get pattern for ANY language (unified interface)
mandatory_pattern() {
    local product lang
    product=$(_mandatory_product_key "$1")
    lang=$(_mandatory_normalize_lang "${2:-go}")

    if [ "$lang" = "java" ]; then
        mandatory_java_pattern "$product"
    else
        _mandatory_generic_pattern "$product" "$lang"
    fi
}

# Show summary of all mandatory tests
mandatory_summary() {
    local product
    echo "$(_mandatory_color blue)=== Mandatory Smoke Test Summary ===$(_mandatory_color reset)"
    echo ""

    while IFS= read -r product; do
        echo "$(_mandatory_color green)▶ Product: $product$(_mandatory_color reset)"
        while IFS= read -r lang; do
            local count
            count=$(_mandatory_generic_count "$product" "$lang")
            printf "   %-8s → %3d tests\n" "$lang" "$count"
        done < <(mandatory_list_languages "$product")
        echo ""
    done < <(mandatory_list_products)
}

# Show help / usage
mandatory_help() {
    cat << 'EOF'
$(_mandatory_color blue)=== Mandatory Smoke Test Help ===$(_mandatory_color reset)

USAGE:
  mandatory_<lang>_pattern <product>     Get test pattern for specific language
  mandatory_pattern <product> [lang]     Unified interface (default: go)
  mandatory_<lang>_count <product>       Count tests for a language
  mandatory_count <product> [lang]       Count tests (unified)

  mandatory_list_products                List all available products
  mandatory_list_languages <product>     List languages for a product
  mandatory_summary                      Show full summary table
  mandatory_help                         Show this help message

LANGUAGES:
  go, python, py, php, node, nodejs, javascript, java

EXAMPLES:
  mandatory_go_pattern "payment_gateway"
  mandatory_pattern "widget" "python"
  mandatory_count "disbursement" "java"
  mandatory_summary

ENVIRONMENT VARIABLES:
  GO_RUNNERS_DIR, NODE_RUNNERS_DIR, etc.  Path to runners directory
  MANDATORY_TESTS_JSON                    Override path to JSON config file
  MANDATORY_UPTIME_SCHEDULE=true          Enable uptime schedule (Go only)
EOF
}

# ---------------------------------------------------------------------------
#  Auto-show summary when sourced interactively (optional convenience)
# ---------------------------------------------------------------------------
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    mandatory_help
fi
