#!/usr/bin/env bash

run_capture_stdout() {
	local expected_status="$1"
	local output_file="$2"
	shift 2

	set +e
	"$@" >"$output_file"
	local status=$?
	set -e

	if [[ "$status" -ne "$expected_status" ]]; then
		printf 'expected exit %s, got %s\n' "$expected_status" "$status" >&2
		if [[ -s "$output_file" ]]; then
			printf 'stdout:\n' >&2
			cat "$output_file" >&2
		fi
		return 1
	fi
}

run_capture_all() {
	local expected_status="$1"
	local output_file="$2"
	shift 2

	set +e
	"$@" >"$output_file" 2>&1
	local status=$?
	set -e

	if [[ "$status" -ne "$expected_status" ]]; then
		printf 'expected exit %s, got %s\n' "$expected_status" "$status" >&2
		if [[ -s "$output_file" ]]; then
			printf 'output:\n' >&2
			cat "$output_file" >&2
		fi
		return 1
	fi
}

assert_jq() {
	local expression="$1"
	local input_file="$2"
	jq -e "$expression" "$input_file" >/dev/null
}

assert_contains() {
	local needle="$1"
	local input_file="$2"
	grep -Fq "$needle" "$input_file"
}
