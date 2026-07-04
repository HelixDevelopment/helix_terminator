#!/usr/bin/env bash
#
# regenerate_exports.sh
#
# Regenerates the multi-format exports for the docs corpus:
#   - Markdown  (docs/markdown/*.md)      -> HTML + DOCX + PDF (docs/html, docs/docx, docs/pdf)
#   - Mermaid   (diagrams/mermaid/*.mmd)  -> SVG + PNG (+ best-effort PDF)
#
# Never overwrites an existing export: if the target file already exists it is
# skipped (SKIP), making the script safe to re-run and safe to point at the
# real, committed output/ tree without clobbering anything. Point --out-dir at
# a scratch mirror to force full regeneration.
#
# Usage:
#   regenerate_exports.sh [--out-dir DIR] [--only md|mermaid]
#
set -euo pipefail

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

REPO_ROOT="$(git rev-parse --show-toplevel)"

export PATH="$HOME/Factory/software/pandoc/bin:$HOME/Factory/software/weasyprint/bin:$HOME/.nvm/versions/node/v24.18.0/bin:$PATH"

ROOT_DIR="$REPO_ROOT/docs/research/mvp/output"
ONLY="all"

usage() {
    echo "Usage: $0 [--out-dir DIR] [--only md|mermaid]" >&2
    exit 1
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --out-dir)
            [[ $# -ge 2 ]] || usage
            ROOT_DIR="$2"
            shift 2
            ;;
        --only)
            [[ $# -ge 2 ]] || usage
            ONLY="$2"
            case "$ONLY" in
                md|mermaid) ;;
                *) echo "ERROR: --only must be 'md' or 'mermaid'" >&2; exit 1 ;;
            esac
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "ERROR: unknown argument: $1" >&2
            usage
            ;;
    esac
done

MD_SRC_DIR="$ROOT_DIR/docs/markdown"
HTML_DIR="$ROOT_DIR/docs/html"
DOCX_DIR="$ROOT_DIR/docs/docx"
PDF_DIR="$ROOT_DIR/docs/pdf"

MMD_SRC_DIR="$ROOT_DIR/diagrams/mermaid"
SVG_DIR="$MMD_SRC_DIR/svg"
PNG_DIR="$MMD_SRC_DIR/png"
MMD_PDF_DIR="$MMD_SRC_DIR/pdf"

for tool in pandoc weasyprint mmdc; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        echo "ERROR: required tool not found on PATH: $tool" >&2
        exit 1
    fi
done

# ---------------------------------------------------------------------------
# Counters
# ---------------------------------------------------------------------------

count_ok_html=0;  count_skip_html=0;  count_fail_html=0
count_ok_docx=0;  count_skip_docx=0;  count_fail_docx=0
count_ok_pdf=0;   count_skip_pdf=0;   count_fail_pdf=0
count_ok_svg=0;   count_skip_svg=0;   count_fail_svg=0
count_ok_png=0;   count_skip_png=0;   count_fail_png=0
count_ok_mpdf=0;  count_skip_mpdf=0;  count_warn_mpdf=0

failures=()

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# run_conversion <label> <out_file> <counter_prefix> <hard_fail:0|1> -- <cmd...>
# Skips if out_file exists. Otherwise runs the command; on success reports OK,
# on failure reports FAIL (and, if hard_fail=1, records it for the final
# non-zero exit).
run_conversion() {
    local label="$1" out_file="$2" prefix="$3" hard_fail="$4"
    shift 4
    if [[ "$1" == "--" ]]; then shift; fi

    if [[ -e "$out_file" ]]; then
        echo "SKIP  $label (exists: $out_file)"
        eval "count_skip_${prefix}=\$((count_skip_${prefix} + 1))"
        return 0
    fi

    local stderr_file
    stderr_file="$(mktemp)"
    if "$@" >/dev/null 2>"$stderr_file"; then
        echo "OK    $label -> $out_file"
        eval "count_ok_${prefix}=\$((count_ok_${prefix} + 1))"
        rm -f "$stderr_file"
        return 0
    else
        local reason
        reason="$(tail -n 3 "$stderr_file" | tr '\n' ' ')"
        rm -f "$stderr_file"
        if [[ "$hard_fail" == "1" ]]; then
            echo "FAIL  $label: $reason"
            eval "count_fail_${prefix}=\$((count_fail_${prefix} + 1))"
            failures+=("$label")
        else
            echo "FAIL  $label (best-effort): $reason"
            eval "count_warn_${prefix}=\$((count_warn_${prefix} + 1))"
        fi
        return 1
    fi
}

# ---------------------------------------------------------------------------
# Markdown -> html / docx / pdf
# ---------------------------------------------------------------------------

process_markdown() {
    if [[ ! -d "$MD_SRC_DIR" ]]; then
        echo "ERROR: markdown source dir not found: $MD_SRC_DIR" >&2
        exit 1
    fi

    mkdir -p "$HTML_DIR" "$DOCX_DIR" "$PDF_DIR"

    shopt -s nullglob
    local md_files=("$MD_SRC_DIR"/*.md)
    shopt -u nullglob

    if [[ ${#md_files[@]} -eq 0 ]]; then
        echo "WARN  no markdown files found in $MD_SRC_DIR"
        return 0
    fi

    local md base html_out docx_out pdf_out
    for md in "${md_files[@]}"; do
        base="$(basename "$md" .md)"
        html_out="$HTML_DIR/${base}.html"
        docx_out="$DOCX_DIR/${base}.docx"
        pdf_out="$PDF_DIR/${base}.pdf"

        # NOTE: -f markdown-yaml_metadata_block disables pandoc's YAML metadata
        # block parsing. Several source docs contain leading/embedded `---`
        # horizontal-rule separators that pandoc would otherwise try (and fail)
        # to parse as a YAML front-matter block. Disabling it is harmless for
        # docs that have no such block.
        run_conversion "$base.md -> html" "$html_out" "html" 1 -- \
            pandoc -f markdown-yaml_metadata_block "$md" -o "$html_out" || true

        run_conversion "$base.md -> docx" "$docx_out" "docx" 1 -- \
            pandoc -f markdown-yaml_metadata_block "$md" -o "$docx_out" || true

        run_conversion "$base.md -> pdf" "$pdf_out" "pdf" 1 -- \
            pandoc -f markdown-yaml_metadata_block "$md" -t html --pdf-engine=weasyprint -o "$pdf_out" || true
    done
}

# ---------------------------------------------------------------------------
# Mermaid -> svg / png / pdf (best-effort)
# ---------------------------------------------------------------------------

process_mermaid() {
    if [[ ! -d "$MMD_SRC_DIR" ]]; then
        echo "ERROR: mermaid source dir not found: $MMD_SRC_DIR" >&2
        exit 1
    fi

    mkdir -p "$SVG_DIR" "$PNG_DIR" "$MMD_PDF_DIR"

    shopt -s nullglob
    local mmd_files=("$MMD_SRC_DIR"/*.mmd)
    shopt -u nullglob

    if [[ ${#mmd_files[@]} -eq 0 ]]; then
        echo "WARN  no mermaid files found in $MMD_SRC_DIR"
        return 0
    fi

    local mmd base svg_out png_out pdf_out
    for mmd in "${mmd_files[@]}"; do
        base="$(basename "$mmd" .mmd)"
        svg_out="$SVG_DIR/${base}.svg"
        png_out="$PNG_DIR/${base}.png"
        pdf_out="$MMD_PDF_DIR/${base}.pdf"

        run_conversion "$base.mmd -> svg" "$svg_out" "svg" 1 -- \
            mmdc -i "$mmd" -o "$svg_out" || true

        run_conversion "$base.mmd -> png" "$png_out" "png" 1 -- \
            mmdc -i "$mmd" -o "$png_out" || true

        # PDF is best-effort: failures here are warnings, not hard failures.
        run_conversion "$base.mmd -> pdf" "$pdf_out" "mpdf" 0 -- \
            mmdc -i "$mmd" -o "$pdf_out" || true
    done
}

# ---------------------------------------------------------------------------
# Run
# ---------------------------------------------------------------------------

echo "== docs export regeneration =="
echo "repo root : $REPO_ROOT"
echo "root dir  : $ROOT_DIR"
echo "selector  : $ONLY"
echo

case "$ONLY" in
    all)
        process_markdown
        echo
        process_mermaid
        ;;
    md)
        process_markdown
        ;;
    mermaid)
        process_mermaid
        ;;
esac

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

echo
echo "== summary =="
if [[ "$ONLY" == "all" || "$ONLY" == "md" ]]; then
    echo "html : ok=$count_ok_html skip=$count_skip_html fail=$count_fail_html"
    echo "docx : ok=$count_ok_docx skip=$count_skip_docx fail=$count_fail_docx"
    echo "pdf  : ok=$count_ok_pdf skip=$count_skip_pdf fail=$count_fail_pdf"
fi
if [[ "$ONLY" == "all" || "$ONLY" == "mermaid" ]]; then
    echo "svg  : ok=$count_ok_svg skip=$count_skip_svg fail=$count_fail_svg"
    echo "png  : ok=$count_ok_png skip=$count_skip_png fail=$count_fail_png"
    echo "mpdf : ok=$count_ok_mpdf skip=$count_skip_mpdf warn(best-effort)=$count_warn_mpdf"
fi

total_fail=${#failures[@]}
if [[ $total_fail -gt 0 ]]; then
    echo
    echo "FAILURES ($total_fail):"
    for f in "${failures[@]}"; do
        echo "  - $f"
    done
    echo
    echo "RESULT: FAIL"
    exit 1
else
    echo
    echo "RESULT: OK"
    exit 0
fi
