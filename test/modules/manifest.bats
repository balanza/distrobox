#!/usr/bin/env bats

setup() {
  # Set the test root as the project root
  DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/../.." >/dev/null 2>&1 && pwd)"
}

trace() {
  #  while IFS= read -r line; do
  #    echo "$line" >&3
  #  done <<<"$1"

  echo "$1" >&3
}

trace_cat() {
  cat "$1" >&3,
}

@test "should resolve manifest file with an include" {
  . "$DIR/modules/manifest"

  local input_file=$(mktemp -u)
  local output_file=$(mktemp -u)

  cat <<EOF >"$input_file"
[box]
image=ubuntu
additional_packages="git"
  
[rootbox]
include=box
root=true
EOF

  expected=$'[box]
image=ubuntu
additional_packages="git"
  
[rootbox]
image=ubuntu
additional_packages="git"
root=true'

  run manifest_resolve_includes "$input_file" "$output_file"

  [ "$status" -eq 0 ]
  [ "$output" = "$output_file" ]

  content=$(cat "$output_file")
  [ "$content" = "$expected" ]
}

@test "should resolve manifest file with a nested include" {
  . "$DIR/modules/manifest"

  local input_file=$(mktemp -u)
  local output_file=$(mktemp -u)

  cat <<EOF >"$input_file"
[box]
image=ubuntu
additional_packages="git"
  
[rootbox]
include=box
root=true

[fullbox]
include=rootbox
init_hooks="echo hi"
EOF

  expected=$'[box]
image=ubuntu
additional_packages="git"
  
[rootbox]
image=ubuntu
additional_packages="git"
root=true

[fullbox]
image=ubuntu
additional_packages="git"
root=true
init_hooks="echo hi"'

  run manifest_resolve_includes "$input_file" "$output_file"

  [ "$status" -eq 0 ]
  [ "$output" = "$output_file" ]

  content=$(cat "$output_file")
  [ "$content" = "$expected" ]
}

@test "should resolve manifest file with no include" {
  . "$DIR/modules/manifest"

  input_file=$(mktemp -u)
  output_file=$(mktemp -u)

  cat <<EOF >"$input_file"
[my-box]
image=ubuntu
additional_packages="git"
  
[my-other-box]
image=fedora
root=true
EOF

  run manifest_resolve_includes "$input_file" "$output_file"
  [ "$status" -eq 0 ]
  [ "$output" = "$output_file" ]

  run cmp -s "$input_file" "$output_file"
  [ "$status" -eq 0 ]

}

@test "should not resolve manifest file with an unknown include" {
  . "$DIR/modules/manifest"

  local input_file=$(mktemp -u)
  local output_file=$(mktemp -u)

  cat <<EOF >"$input_file"
[box]
image=ubuntu
additional_packages="git"
  
[rootbox]
include=unknown
root=true
EOF

  run manifest_resolve_includes "$input_file" "$output_file"

  [ "$status" -ne 0 ]

}

@test "should not resolve manifest file with circular references" {
  . "$DIR/modules/manifest"

  local input_file=$(mktemp -u)
  local output_file=$(mktemp -u)

  cat <<EOF >"$input_file"
[box]
image=ubuntu
include=finalbox
  
[rootbox]
include=box
root=true

[finalbox]
include=rootbox
additional_packages="git"
EOF

  run manifest_resolve_includes "$input_file" "$output_file"

  [ "$status" -ne 0 ]
  trace "$output"
}

@test "should not resolve manifest file with self references" {
  . "$DIR/modules/manifest"

  local input_file=$(mktemp -u)
  local output_file=$(mktemp -u)

  cat <<EOF >"$input_file"
[dannunzio]
include=dannunzio
image=ubuntu
EOF

  run manifest_resolve_includes "$input_file" "$output_file"

  [ "$status" -ne 0 ]
  trace "$output"
}

@test "should extract manifest section on the top of the file" {
  . "$DIR/modules/manifest"

  input_file=$(mktemp -u)

  cat <<EOF >"$input_file"
[my-box]
image=ubuntu
additional_packages="git"
  
[my-other-box]
image=fedora
root=true

[my-final-box]
image=leap
additional_packages="nvim"
EOF

  expected=$'image=ubuntu
additional_packages="git"'

  run manifest_read_section "my-box" "$input_file"

  [ "$status" -eq 0 ]
  [ "$output" = "$expected" ]

}

@test "should extract manifest section in the middle of the file" {
  . "$DIR/modules/manifest"

  input_file=$(mktemp -u)

  cat <<EOF >"$input_file"
[my-box]
image=ubuntu
additional_packages="git"
  
[my-other-box]
image=fedora
root=true

[my-final-box]
image=leap
additional_packages="nvim"
EOF

  expected=$'image=fedora
root=true'

  run manifest_read_section "my-other-box" "$input_file"

  [ "$status" -eq 0 ]
  [ "$output" = "$expected" ]

}

@test "should extract manifest section on the bottom of the file" {
  . "$DIR/modules/manifest"

  input_file=$(mktemp -u)

  cat <<EOF >"$input_file"
[my-box]
image=ubuntu
additional_packages="git"
  
[my-other-box]
image=fedora
root=true

[my-final-box]
image=leap
additional_packages="nvim"
EOF

  expected=$'image=leap
additional_packages="nvim"'

  run manifest_read_section "my-final-box" "$input_file"

  [ "$status" -eq 0 ]
  [ "$output" = "$expected" ]

}

@test "should replace in the same file" {
  . "$DIR/modules/manifest"

  input_file=$(mktemp -u)

  cat <<EOF >"$input_file"
line1
line2
line3
EOF

  to_add=$'line4
line5'

  expected=$'line1
line4
line5
line3'

  run manifest_replace_line 2 "$to_add" "$input_file" "$input_file"

  content=$(cat "$input_file")

  [ "$status" -eq 0 ]
  [ "$output" = "$input_file" ]
  [ "$content" = "$expected" ]

}

@test "should replace in new file" {
  . "$DIR/modules/manifest"

  input_file=$(mktemp -u)

  cat <<EOF >"$input_file"
line1
line2
line3
EOF

  to_add=$'line4
line5'

  expected=$'line1
line4
line5
line3'

  run manifest_replace_line 2 "$to_add" "$input_file"

  content=$(cat "$output")

  [ "$status" -eq 0 ]
  [ "$content" = "$expected" ]

}
