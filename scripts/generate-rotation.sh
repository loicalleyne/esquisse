#!/usr/bin/env bash

set -euo pipefail

die() {
	printf 'generate-rotation: %s\n' "$1" >&2
	exit 1
}

usage() {
	cat <<'EOF'
Usage: generate-rotation.sh [--models PATH] [--repo-root PATH] [--dry-run]

Defaults:
	--models    ~/.esquisse/models.available.json
	--repo-root .
EOF
}

# TSV column order: vendor, id, name, family, key, score
row_vendor() { IFS=$'\t' read -r vendor _              <<< "$1"; printf '%s' "$vendor"; }
row_id()     { IFS=$'\t' read -r _ id _                <<< "$1"; printf '%s' "$id";     }
row_name()   { IFS=$'\t' read -r _ _ name _             <<< "$1"; printf '%s' "$name";   }
row_family() { IFS=$'\t' read -r _ _ _ family _         <<< "$1"; printf '%s' "$family"; }
row_key()    { IFS=$'\t' read -r _ _ _ _ key _          <<< "$1"; printf '%s' "$key";    }
row_score()  { IFS=$'\t' read -r _ _ _ _ _ score        <<< "$1"; printf '%s' "$score";  }

row_label() {
	local vendor name
	vendor="$(row_vendor "$1")"
	name="$(row_name "$1")"
	printf '%s (%s)' "$name" "$vendor"
}

update_if_changed() {
	local file_path="$1"
	local tmp_before
	tmp_before="$(mktemp)"
	cp "$file_path" "$tmp_before"

	shift
	"$@"

	if cmp -s "$tmp_before" "$file_path"; then
		rm -f "$tmp_before"
		return 1
	fi

	if [ "$DRY_RUN" -eq 1 ]; then
		cp "$tmp_before" "$file_path"
	fi

	rm -f "$tmp_before"
	return 0
}

update_agent_file() {
	local file_path="$1"
	local model0="$2"
	local model1="$3"
	local model2="$4"

	MODEL0="$model0" MODEL1="$model1" MODEL2="$model2" perl -0pi -e '
		my $m0 = $ENV{MODEL0};
		my $m1 = $ENV{MODEL1};
		my $m2 = $ENV{MODEL2};
		my $ok = 1;
		$ok &&= s/^model:\s*\[.*\]$/model: [\x27$m0\x27, \x27$m1\x27, \x27$m2\x27]/m;
		die "Could not find model line in agent file\n" unless $ok;
	' "$file_path"
}

update_planner_file() {
	local file_path="$1"
	local slot0="$2"
	local slot1="$3"
	local slot2="$4"

	# update model: frontmatter line (same as adversarial agent files)
	update_agent_file "$file_path" "$slot0" "$slot1" "$slot2"

	# update slot description lines in the body (best-effort; warn on mismatch)
	SLOT0="$slot0" SLOT1="$slot1" SLOT2="$slot2" perl -0pi -e '
		my $s0 = $ENV{SLOT0};
		my $s1 = $ENV{SLOT1};
		my $s2 = $ENV{SLOT2};
		s/- slot 0 \x{2192} `\@Adversarial-r0` \([^\n]*\)/- slot 0 \x{2192} `\@Adversarial-r0` ($s0)/;
		s/- slot 1 \x{2192} `\@Adversarial-r1` \([^\n]*\)/- slot 1 \x{2192} `\@Adversarial-r1` ($s1)/;
		s/- slot 2 \x{2192} `\@Adversarial-r2` \([^\n]*\)/- slot 2 \x{2192} `\@Adversarial-r2` ($s2)/;
	' "$file_path"
}

update_crush_file() {
	local file_path="$1"
	local slot0="$2"
	local slot1="$3"
	local slot2="$4"

	CRUSH0="$slot0" CRUSH1="$slot1" CRUSH2="$slot2" perl -0pi -e '
		my $c0 = $ENV{CRUSH0};
		my $c1 = $ENV{CRUSH1};
		my $c2 = $ENV{CRUSH2};
		my ($n0) = split m{/}, $c0, 2;
		my ($n1) = split m{/}, $c1, 2;
		my ($n2) = split m{/}, $c2, 2;
		$n0 = $c0 unless defined $n0 && length $n0;
		$n1 = $c1 unless defined $n1 && length $n1;
		$n2 = $c2 unless defined $n2 && length $n2;
		my $ok = 1;
		$ok &&= s/\| 0 \| 0 \| `[^`]+` \|/| 0 | 0 | `$c0` |/;
		$ok &&= s/\| 1 \| 1 \| `[^`]+` \|/| 1 | 1 | `$c1` |/;
		$ok &&= s/\| 2 \| 2 \| `[^`]+` \|/| 2 | 2 | `$c2` |/;
		$ok &&= s{These mirror the VS Code rotation \(Adversarial-r0 = .*?self-review bias\.}{These mirror the VS Code rotation (Adversarial-r0 = $n0, r1 = $n1, r2 = $n2). The models are deliberately cross-provider to prevent\nself-review bias.}s;
		die "Could not update crush model reference\n" unless $ok;
	' "$file_path"
}

MODELS_PATH="$HOME/.esquisse/models.available.json"
REPO_ROOT="."
DRY_RUN=0

while [ "$#" -gt 0 ]; do
	case "$1" in
		--models)
			shift
			[ "$#" -gt 0 ] || die "--models requires a path"
			MODELS_PATH="$1"
			;;
		--repo-root)
			shift
			[ "$#" -gt 0 ] || die "--repo-root requires a path"
			REPO_ROOT="$1"
			;;
		--dry-run)
			DRY_RUN=1
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			die "unknown argument: $1"
			;;
	esac
	shift
done

command -v jq >/dev/null 2>&1 || die "jq is required"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$REPO_ROOT" && pwd)"

[ -f "$MODELS_PATH" ] || die "model snapshot not found: $MODELS_PATH"
[ -f "$REPO_ROOT/.github/agents/EsquissePlan.agent.md" ] || die "missing target repo files under: $REPO_ROOT"

rows_tmp="$(mktemp)"
RAND_SEED=$RANDOM
trap 'rm -f "$rows_tmp"' EXIT

jq -r '
	def source_models:
		if type == "object" and has("models") then .models
		elif type == "array" then .
		else error("Input JSON must be either { models: [...] } or an array of models")
		end;
	def norm: ascii_downcase;
	def key: "\((.vendor // "") | norm)/\((.id // .name // "") | norm)";
	# broad family = drop all dash-segments that start with a digit (strips version numbers)
	def family:
		(.family // .id // "unknown") | ascii_downcase
		| split("-") | map(select(test("^[^0-9]"))) | join("-")
		| if . == "" then "unknown" else . end;
	def score:
		(key) as $k |
		0
		+ (if ($k|contains("opus")) then 120 else 0 end)
		+ (if ($k|contains("sonnet")) then 110 else 0 end)
		+ (if ($k|contains("gpt-5")) then 100 else 0 end)
		+ (if ($k|contains("gpt-4.1")) then 95 else 0 end)
		+ (if ($k|contains("gpt-4o")) then 90 else 0 end)
		+ (if ($k|contains("o1")) then 85 else 0 end)
		+ (if ($k|contains("gemini-2.5-pro")) then 88 else 0 end)
		+ (if (($k|contains("gemini")) and ($k|contains("pro"))) then 80 else 0 end)
		+ (if (($k|contains("mini")) or ($k|contains("flash")) or ($k|contains("haiku"))) then -30 else 0 end)
		+ (if (.vendor // "" | norm) == "copilot" then 10 else 0 end);
	source_models
	| .[]
	| [(.vendor // "copilot" | tostring), (.id // .name // "" | tostring), (.name // .id // "" | tostring), family, key, (score|tostring)]
	| @tsv
' "$MODELS_PATH" | awk -F '\t' -v seed="$RAND_SEED" 'BEGIN{srand(seed)} NF && !seen[$5]++ { $6=$6+int(rand()*20); print }' OFS='\t' > "$rows_tmp"

rows=()
while IFS= read -r row; do
	rows+=("$row")
done < <(sort -t "$(printf '\t')" -k6,6nr "$rows_tmp")

[ "${#rows[@]}" -ge 3 ] || die "Need at least 3 distinct models; found ${#rows[@]}"

# Pick 3 models maximising family diversity first, then vendor diversity.
# Pass 1: one model per distinct family (highest-scored within family).
# Pass 2: if still < 3, fill with different vendors.
# Pass 3: if still < 3, fill with any remaining model.
selected_rows=()
selected_families=()
selected_vendors=()

is_selected() {
	local r="$1"
	for e in "${selected_rows[@]+${selected_rows[@]}}"; do
		[ "$r" = "$e" ] && return 0
	done
	return 1
}

# EXCLUDE_ROWS (global): rows to skip entirely; sets FILL_ROWS with COUNT diverse rows
pick_diverse() {
	local count="$1"
	FILL_ROWS=()
	local used_families=() used_vendors=() fam v skip row e

	for row in "${rows[@]}"; do
		[ "${#FILL_ROWS[@]}" -ge "$count" ] && break
		skip=0; for e in "${EXCLUDE_ROWS[@]+${EXCLUDE_ROWS[@]}}"; do [ "$row" = "$e" ] && skip=1 && break; done
		[ "$skip" -eq 1 ] && continue
		fam="$(row_family "$row")"; skip=0
		for f in "${used_families[@]+${used_families[@]}}"; do [ "$fam" = "$f" ] && skip=1 && break; done
		[ "$skip" -eq 1 ] && continue
		FILL_ROWS+=("$row"); used_families+=("$fam"); used_vendors+=("$(row_vendor "$row")")
	done

	for row in "${rows[@]}"; do
		[ "${#FILL_ROWS[@]}" -ge "$count" ] && break
		skip=0; for e in "${EXCLUDE_ROWS[@]+${EXCLUDE_ROWS[@]}}"; do [ "$row" = "$e" ] && skip=1 && break; done
		[ "$skip" -eq 1 ] && continue
		skip=0; for e in "${FILL_ROWS[@]+${FILL_ROWS[@]}}"; do [ "$row" = "$e" ] && skip=1 && break; done
		[ "$skip" -eq 1 ] && continue
		v="$(row_vendor "$row")"; skip=0
		for sv in "${used_vendors[@]+${used_vendors[@]}}"; do [ "$v" = "$sv" ] && skip=1 && break; done
		[ "$skip" -eq 1 ] && continue
		FILL_ROWS+=("$row"); used_vendors+=("$v")
	done

	for row in "${rows[@]}"; do
		[ "${#FILL_ROWS[@]}" -ge "$count" ] && break
		skip=0; for e in "${EXCLUDE_ROWS[@]+${EXCLUDE_ROWS[@]}}"; do [ "$row" = "$e" ] && skip=1 && break; done
		[ "$skip" -eq 1 ] && continue
		skip=0; for e in "${FILL_ROWS[@]+${FILL_ROWS[@]}}"; do [ "$row" = "$e" ] && skip=1 && break; done
		[ "$skip" -eq 1 ] && continue
		FILL_ROWS+=("$row")
	done
}

# Pass 1: distinct families
for row in "${rows[@]}"; do
	[ "${#selected_rows[@]}" -ge 3 ] && break
	is_selected "$row" && continue
	fam="$(row_family "$row")"
	for f in "${selected_families[@]+${selected_families[@]}}"; do
		[ "$fam" = "$f" ] && fam=""
	done
	[ -z "$fam" ] && continue
	selected_rows+=("$row")
	selected_families+=("$(row_family "$row")")
	selected_vendors+=("$(row_vendor "$row")")
done

# Pass 2: distinct vendors (catches same-family different-vendor as last resort)
for row in "${rows[@]}"; do
	[ "${#selected_rows[@]}" -ge 3 ] && break
	is_selected "$row" && continue
	v="$(row_vendor "$row")"
	for sv in "${selected_vendors[@]+${selected_vendors[@]}}"; do
		[ "$v" = "$sv" ] && v=""
	done
	[ -z "$v" ] && continue
	selected_rows+=("$row")
	selected_families+=("$(row_family "$row")")
	selected_vendors+=("$(row_vendor "$row")")
done

# Pass 3: any remaining model
for row in "${rows[@]}"; do
	[ "${#selected_rows[@]}" -ge 3 ] && break
	is_selected "$row" && continue
	selected_rows+=("$row")
done

[ "${#selected_rows[@]}" -ge 3 ] || die "Need at least 3 models after selection"

slot0_vendor="$(row_vendor "${selected_rows[0]}")"  ; slot0_id="$(row_id "${selected_rows[0]}")"     ; slot0_name="$(row_name "${selected_rows[0]}")"
slot0_family="$(row_family "${selected_rows[0]}")" ; slot0_label="$(row_label "${selected_rows[0]}")" ; slot0_model="${slot0_vendor}/${slot0_id}"

slot1_vendor="$(row_vendor "${selected_rows[1]}")"  ; slot1_id="$(row_id "${selected_rows[1]}")"     ; slot1_name="$(row_name "${selected_rows[1]}")"
slot1_family="$(row_family "${selected_rows[1]}")" ; slot1_label="$(row_label "${selected_rows[1]}")" ; slot1_model="${slot1_vendor}/${slot1_id}"

slot2_vendor="$(row_vendor "${selected_rows[2]}")"  ; slot2_id="$(row_id "${selected_rows[2]}")"     ; slot2_name="$(row_name "${selected_rows[2]}")"
slot2_family="$(row_family "${selected_rows[2]}")" ; slot2_label="$(row_label "${selected_rows[2]}")" ; slot2_model="${slot2_vendor}/${slot2_id}"

# Planner: 3 diverse models that are NOT any reviewer's primary model
EXCLUDE_ROWS=("${selected_rows[0]}" "${selected_rows[1]}" "${selected_rows[2]}")
pick_diverse 3
planner_rows=("${FILL_ROWS[@]}")
[ "${#planner_rows[@]}" -ge 1 ] || die "No models available for planner outside reviewer primaries"
while [ "${#planner_rows[@]}" -lt 3 ]; do planner_rows+=("${planner_rows[0]}"); done
planner_label0="$(row_label "${planner_rows[0]}")"
planner_label1="$(row_label "${planner_rows[1]}")"
planner_label2="$(row_label "${planner_rows[2]}")"

# Reviewer fills: exclude own primary AND planner primary so consecutive runs never share a model
EXCLUDE_ROWS=("${selected_rows[0]}" "${planner_rows[0]}")
pick_diverse 2
r0_fill0_label="$(row_label "${FILL_ROWS[0]}")"
r0_fill1_label="$([ "${#FILL_ROWS[@]}" -ge 2 ] && row_label "${FILL_ROWS[1]}" || row_label "${FILL_ROWS[0]}")"

EXCLUDE_ROWS=("${selected_rows[1]}" "${planner_rows[0]}")
pick_diverse 2
r1_fill0_label="$(row_label "${FILL_ROWS[0]}")"
r1_fill1_label="$([ "${#FILL_ROWS[@]}" -ge 2 ] && row_label "${FILL_ROWS[1]}" || row_label "${FILL_ROWS[0]}")"

EXCLUDE_ROWS=("${selected_rows[2]}" "${planner_rows[0]}")
pick_diverse 2
r2_fill0_label="$(row_label "${FILL_ROWS[0]}")"
r2_fill1_label="$([ "${#FILL_ROWS[@]}" -ge 2 ] && row_label "${FILL_ROWS[1]}" || row_label "${FILL_ROWS[0]}")"

changed_files=()

if update_if_changed "$REPO_ROOT/.github/agents/EsquissePlan.agent.md" update_planner_file "$REPO_ROOT/.github/agents/EsquissePlan.agent.md" "$planner_label0" "$planner_label1" "$planner_label2"; then
	changed_files+=("$REPO_ROOT/.github/agents/EsquissePlan.agent.md")
fi

if update_if_changed "$REPO_ROOT/.github/agents/Adversarial-r0.agent.md" update_agent_file "$REPO_ROOT/.github/agents/Adversarial-r0.agent.md" "$slot0_label" "$r0_fill0_label" "$r0_fill1_label"; then
	changed_files+=("$REPO_ROOT/.github/agents/Adversarial-r0.agent.md")
fi

if update_if_changed "$REPO_ROOT/.github/agents/Adversarial-r1.agent.md" update_agent_file "$REPO_ROOT/.github/agents/Adversarial-r1.agent.md" "$slot1_label" "$r1_fill0_label" "$r1_fill1_label"; then
	changed_files+=("$REPO_ROOT/.github/agents/Adversarial-r1.agent.md")
fi

if update_if_changed "$REPO_ROOT/.github/agents/Adversarial-r2.agent.md" update_agent_file "$REPO_ROOT/.github/agents/Adversarial-r2.agent.md" "$slot2_label" "$r2_fill0_label" "$r2_fill1_label"; then
	changed_files+=("$REPO_ROOT/.github/agents/Adversarial-r2.agent.md")
fi

if update_if_changed "$REPO_ROOT/skills/adversarial-review/crush-models.md" update_crush_file "$REPO_ROOT/skills/adversarial-review/crush-models.md" "$slot0_model" "$slot1_model" "$slot2_model"; then
	changed_files+=("$REPO_ROOT/skills/adversarial-review/crush-models.md")
fi

if [ "$DRY_RUN" -eq 1 ]; then
	# Dry-run already mutates through temp compare only; no extra action needed.
	:
fi

picked_json="$(jq -n \
	--arg vendor0 "$slot0_vendor" --arg id0 "$slot0_id" --arg name0 "$slot0_name" --arg family0 "$slot0_family" --arg label0 "$slot0_label" \
	--arg vendor1 "$slot1_vendor" --arg id1 "$slot1_id" --arg name1 "$slot1_name" --arg family1 "$slot1_family" --arg label1 "$slot1_label" \
	--arg vendor2 "$slot2_vendor" --arg id2 "$slot2_id" --arg name2 "$slot2_name" --arg family2 "$slot2_family" --arg label2 "$slot2_label" \
	--arg r0f0 "$r0_fill0_label" --arg r0f1 "$r0_fill1_label" \
	--arg r1f0 "$r1_fill0_label" --arg r1f1 "$r1_fill1_label" \
	--arg r2f0 "$r2_fill0_label" --arg r2f1 "$r2_fill1_label" \
	--arg pl0 "$planner_label0" --arg pl1 "$planner_label1" --arg pl2 "$planner_label2" \
	'{
		planner: {models: [$pl0, $pl1, $pl2]},
		reviewers: [
			{slot: 0, vendor: $vendor0, id: $id0, name: $name0, family: $family0, label: $label0, models: [$label0, $r0f0, $r0f1]},
			{slot: 1, vendor: $vendor1, id: $id1, name: $name1, family: $family1, label: $label1, models: [$label1, $r1f0, $r1f1]},
			{slot: 2, vendor: $vendor2, id: $id2, name: $name2, family: $family2, label: $label2, models: [$label2, $r2f0, $r2f1]}
		]
	}'
)"

if [ "${#changed_files[@]}" -gt 0 ]; then
	changed_json="$(printf '%s\n' "${changed_files[@]}" | jq -R . | jq -s .)"
else
	changed_json='[]'
fi

jq -n \
	--arg dry_run "$DRY_RUN" \
	--argjson picked "$picked_json" \
	--argjson changed_files "$changed_json" \
	'{dry_run: ($dry_run == "1"), picked: $picked, changed_files: $changed_files}'
