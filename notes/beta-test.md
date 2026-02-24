$ git version
git version 2.43.0
zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main$ go version
go version go1.26.0 linux/amd64

zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main$ pwd
/home/zepho/Downloads/bizacademy-main
zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main$ cd stave
zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main/stave$ make build
rm -rf internal/contracts/schemas/embedded/*
cp -R schemas/* internal/contracts/schemas/embedded/
rm -rf internal/adapters/input/invariants/builtin/embedded/*
cp -R invariants/* internal/adapters/input/invariants/builtin/embedded/
go build -ldflags "-s -w -X github.com/sufield/stave/cmd/stave/cmd.Version=0.0.1" -o stave ./cmd/stave

zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main/stave$ ./stave
Tip: run `stave init` to scaffold a starter project.
Stave detects infrastructure resources that have remained unsafe for too long,
using only configuration snapshots—no cloud credentials required.
Output is deterministic when --now is set (required for reproducible CI/CD runs).

Recommended Workflow:
  1. validate  - Check inputs are well-formed (run first!)
  2. evaluate  - Detect violations and produce findings
  3. diagnose  - Understand unexpected results

Input Formats:
  --invariants    Directory with YAML invariant definitions (inv.v0.1)
  --observations  Directory with JSON observation snapshots (obs.v0.1)

Output Formats:
  --format json   Machine-readable JSON on commands that support format selection
  --format text   Human-readable summary on commands that support format selection
  --output ...    Global fallback mode (prefer per-command --format)

Logging:
  -v              Increase verbosity (INFO level)
  -vv             Debug verbosity (DEBUG level)
  --log-level     Explicit level: debug|info|warn|error
  --log-format    Format: text|json (default: text)
  --log-file      Write logs to file instead of stderr
  --log-timestamps  Include timestamps (breaks determinism)
  --log-timings   Include timing information (breaks determinism)

Sharing:
  --redact       Redact infrastructure identifiers from output
  --path-mode    Path rendering: base (default, basenames only) or full (absolute paths)

Exit Codes:
  0   Success, no issues
  2   Invalid input or validation failure
  3   Violations found (evaluate) or diagnostics found (diagnose)
  4   Unexpected internal error
  130 Interrupted (SIGINT/Ctrl+C)

Examples:
  # Step 1: Always validate first
  stave validate --invariants ./inv --observations ./obs

  # Step 2: Evaluate with 7-day threshold
  stave evaluate --invariants ./inv --observations ./obs --max-unsafe 7d

  # Step 3: Diagnose unexpected results
  stave diagnose --invariants ./inv --observations ./obs

  # Verbose mode (INFO level logs to stderr)
  stave evaluate --invariants ./inv --observations ./obs -v

  # Debug mode
  stave evaluate --invariants ./inv --observations ./obs -vv

  # JSON logs to file
  stave evaluate --invariants ./inv --observations ./obs --log-format json --log-file run.log

  # Redact identifiers for safe sharing
  stave evaluate --invariants ./inv --observations ./obs --redact

Documentation: See docs/user-docs.md for detailed usage.

Usage:
  stave [command]

Getting Started
  doctor       Check local environment readiness for Stave workflows
  generate     Generate starter artifacts
  init         Initialize a starter Stave project structure
  quickstart   Run first-use scaffold + validate + evaluate in one command
  template     Print copy-paste templates for common workflow steps

Core Evaluation
  check        Validate, evaluate, and diagnose in one command
  diagnose     Diagnose evaluation inputs and results
  evaluate     Evaluate configuration snapshots for safety violations
  explain      Explain how an invariant evaluates and which fields it needs
  fmt          Format invariant and observation files deterministically
  lint         Lint invariant files for design quality
  trace        Trace predicate evaluation for a single invariant against a single resource
  validate     Validate inputs without evaluation
  verify       Compare before/after evaluations to check remediation

Workflow & CI
  ci           CI/CD policy and baseline commands
  context      Named project context commands
  resume       Alias of status focused on continuing where you left off
  snapshot     Snapshot lifecycle commands
  status       Show project context and the next recommended command

Data & Artifacts
  enforce      Generate deterministic enforcement templates from evaluation output
  extractor    Extractor development commands
  graph        Visualize invariant and resource relationships
  ingest       Convert source snapshots to normalized observations
  invariants   Work with invariant definitions
  packs        Inspect built-in invariant packs
  report       Generate a plain-text report from evaluation output

Utilities & Help
  alias        Manage command aliases
  capabilities Print supported input types and version constraints
  completion   Generate the autocompletion script for the specified shell
  config       Configuration commands
  docs         Documentation workflow commands
  fix          Show machine-readable fix plan for a finding
  help         Help about any command
  prompt       Generate LLM prompts from evaluation results
  version      Print version information

Flags:
      --allow-symlink-output   Allow writing output through symlinks (default: refuse)
      --force                  Allow overwriting existing output files
  -h, --help                   help for stave
      --log-file string        Write logs to file (default: stderr)
      --log-format string      Log format: text|json (default "text")
      --log-level string       Log level: debug|info|warn|error (overrides -v)
      --log-timestamps         Include timestamps in logs (breaks determinism)
      --log-timings            Include timing information (breaks determinism)
      --no-color               Disable ANSI colors in output
      --output string          Output format: json or text Resolved default may come from STAVE_* env vars, stave.yaml, user config, or built-in. (default "text")
      --path-mode string       Path rendering in errors/logs: base (basename only) or full (absolute paths) Resolved default may come from STAVE_* env vars, stave.yaml, user config, or built-in. (default "base")
      --quiet                  Suppress output (exit code only) Resolved default may come from STAVE_* env vars, stave.yaml, user config, or built-in.
      --redact                 Redact infrastructure identifiers (bucket names, ARNs, policies) from output Resolved default may come from STAVE_* env vars, stave.yaml, user config, or built-in.
      --require-offline        Assert offline operation: fail if proxy env vars (HTTP_PROXY, HTTPS_PROXY, ALL_PROXY) are set
  -v, --verbose count          Increase verbosity (-v=INFO, -vv=DEBUG)
      --version                version for stave

Use "stave [command] --help" for more information about a command.
No Stave project found in this directory tree. Run `stave init` to create one.

 ./stave version
0.0.1

 make lint
/home/zepho/go/bin/golangci-lint run ./...
Error: can't load config: the Go language version (go1.25) used to build golangci-lint is lower than the targeted Go version (1.26.0)
The command is terminated due to an error: can't load config: the Go language version (go1.25) used to build golangci-lint is lower than the targeted Go version (1.26.0)
make: *** [Makefile:63: lint] Error 3

curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
  sh -s -- -b "$(go env GOPATH)/bin" v1.64.2

curl -sSfL https://golangci-lint.run/install.sh | \
  sh -s -- -b "$(go env GOPATH)/bin" v2.10.1

$(go env GOPATH)/bin/golangci-lint version

make lint
/home/zepho/go/bin/golangci-lint run ./...
cmd/gendocs/main.go:43:23: G703: Path traversal via taint analysis (gosec)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
	                     ^
cmd/gendocs/main.go:56:17: G703: Path traversal via taint analysis (gosec)
			_ = os.Remove(filepath.Join(outDir, e.Name()))
			             ^
cmd/gendocs/main.go:88:13: G705: XSS via taint analysis (gosec)
	fmt.Fprintf(os.Stderr, "Generated CLI reference docs in %s\n", outDir)
	           ^
cmd/gendocs/main.go:237:21: G703: Path traversal via taint analysis (gosec)
	return os.WriteFile(filepath.Join(outDir, "_index.md"), []byte(b.String()), 0o644)
	                   ^
cmd/gendocs/main.go:322:21: G703: Path traversal via taint analysis (gosec)
	return os.WriteFile(filepath.Join(outDir, fileName+".md"), []byte(b.String()), 0o644)
	                   ^
cmd/genrecordings/main.go:221:23: G703: Path traversal via taint analysis (gosec)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
	                     ^
cmd/genrecordings/main.go:229:33: G703: Path traversal via taint analysis (gosec)
	if existing, err := os.ReadFile(checksumFile); err == nil {
	                               ^
cmd/genrecordings/main.go:246:24: G703: Path traversal via taint analysis (gosec)
	if err := os.WriteFile(checksumFile, []byte(newChecksum+"\n"), 0o644); err != nil {
	                      ^
cmd/genrecordings/main.go:251:13: G705: XSS via taint analysis (gosec)
	fmt.Fprintf(os.Stderr, "Generated %d recordings in %s\n", len(scenarios), outputDir)
	           ^
cmd/genrecordings/main.go:272:21: G702: Command injection via taint analysis (gosec)
	cmd := exec.Command(binary, args...)
	                   ^
cmd/genrecordings/main.go:304:22: G702: Command injection via taint analysis (gosec)
		cmd := exec.Command(binary, st.Args...)
		                   ^
cmd/genrecordings/main.go:316:21: G703: Path traversal via taint analysis (gosec)
	f, err := os.Create(path)
	                   ^
cmd/genrecordings/main.go:392:22: G703: Path traversal via taint analysis (gosec)
	if f, err := os.Open(binary); err == nil {
	                    ^
cmd/stave/cmd/doctor.go:283:15: G703: Path traversal via taint analysis (gosec)
	_ = os.Remove(name)
	             ^
cmd/stave/cmd/doctor.go:337:22: G703: Path traversal via taint analysis (gosec)
	info, err := os.Stat(path)
	                    ^
internal/pathinfer/pathinfer.go:20:21: G703: Path traversal via taint analysis (gosec)
		fi, err := os.Stat(root)
		                  ^
cmd/stave/cmd/hygiene.go:351:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("Generated at: `%s`\n\n", now.Format(time.RFC3339)))
	^
cmd/stave/cmd/hygiene.go:352:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("Trend window: `%s` to `%s` (%s)\n\n",
	^
cmd/stave/cmd/hygiene.go:361:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Total snapshots | %d |\n", snap.Total))
	^
cmd/stave/cmd/hygiene.go:362:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Active snapshots | %d |\n", snap.Active))
	^
cmd/stave/cmd/hygiene.go:363:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Archived snapshots | %d |\n", snap.Archived))
	^
cmd/stave/cmd/hygiene.go:364:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Prune candidates (current) | %d |\n", snap.PruneCandidates))
	^
cmd/stave/cmd/hygiene.go:365:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Retention tier | %s |\n", snap.RetentionTier))
	^
cmd/stave/cmd/hygiene.go:366:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Retention window | %s |\n", domain.FormatDuration(snap.RetentionDuration)))
	^
cmd/stave/cmd/hygiene.go:367:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Keep minimum | %d |\n", snap.KeepMin))
	^
cmd/stave/cmd/hygiene.go:373:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Current violations | %d |\n", risk.CurrentViolations))
	^
cmd/stave/cmd/hygiene.go:374:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Upcoming overdue | %d |\n", risk.Overdue))
	^
cmd/stave/cmd/hygiene.go:375:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Upcoming due now | %d |\n", risk.DueNow))
	^
cmd/stave/cmd/hygiene.go:376:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Upcoming due soon (<= %s) | %d |\n", domain.FormatDurationHuman(dueSoon), risk.DueSoon))
	^
cmd/stave/cmd/hygiene.go:377:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Upcoming later | %d |\n", risk.Later))
	^
cmd/stave/cmd/hygiene.go:378:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("| Upcoming total | %d |\n", risk.UpcomingTotal))
	^
cmd/stave/cmd/hygiene.go:386:3: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
		b.WriteString(fmt.Sprintf(
		^
cmd/stave/cmd/upcoming.go:403:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("- Overdue: **%d**\n", summary.Overdue))
	^
cmd/stave/cmd/upcoming.go:404:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("- Due now: **%d**\n", summary.DueNow))
	^
cmd/stave/cmd/upcoming.go:405:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("- Due soon (<= %s): **%d**\n", domain.FormatDurationHuman(dueSoonThreshold), summary.DueSoon))
	^
cmd/stave/cmd/upcoming.go:406:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("- Later: **%d**\n", summary.Later))
	^
cmd/stave/cmd/upcoming.go:407:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("- Total action items: **%d**\n", summary.Total))
	^
cmd/stave/cmd/upcoming.go:415:2: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
	b.WriteString(fmt.Sprintf("Generated at: `%s`\n\n", opts.Now.UTC().Format(time.RFC3339)))
	^
cmd/stave/cmd/upcoming.go:427:3: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
		b.WriteString(fmt.Sprintf(
		^
39 issues:
* gosec: 16
* staticcheck: 23
make: *** [Makefile:63: lint] Error 1

zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main/stave$ pwd
/home/zepho/Downloads/bizacademy-main/stave
zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main/stave$ mkdir beta-test
zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main/stave$ cd beta-test
zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main/stave/beta-test$ stave init --profile mvp1-s3
Command 'stave' not found, did you mean:
  command 'save' from deb atfs (1.4pl6-16)
Try: sudo apt install <deb name>
zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main/stave/beta-test$ ../stave init --profile mvp1-s3
No git repository found in .. Initialize now? [Y/n]: Y
Initialized git repository at .
Initialized empty Stave project in /home/zepho/Downloads/bizacademy-main/stave/beta-test

Created structure:
  - invariants/
  - snapshots/raw/
  - observations/
  - output/
  - snapshots/raw/aws-s3/
  - cli.yaml
  - stave.lock
  - stave.yaml
  - snapshots/raw/aws-s3/README.md
  - snapshots/raw/.gitkeep
  - invariants/.gitkeep
  - README.md
  - observations/2026-01-11T00:00:00Z.json
  - observations/2026-01-18T00:00:00Z.json
  - snapshots/raw/aws-s3/.gitkeep
  - output/.gitkeep
  - .gitignore

Next steps:
  1. Run `stave doctor` to verify your local environment.
  2. Run `stave evaluate --observations ./observations` to evaluate with built-in S3 checks.
  3. Run `stave snapshot upcoming --observations ./observations` to see the next snapshot schedule.
  4. Read the generated README.md for the full recommended workflow.
Next workflow start: stave template --step invariant > /home/zepho/Downloads/bizacademy-main/stave/beta-test/invariants/INV.S3.PUBLIC.901.yaml
zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main/stave/beta-test$ ll
total 52
drwxrwxr-x  8 zepho zepho 4096 Feb 24 11:48 ./
drwxrwxr-x 15 zepho zepho 4096 Feb 24 11:41 ../
-rw-r--r--  1 zepho zepho  468 Feb 24 11:48 cli.yaml
drwxrwxr-x  7 zepho zepho 4096 Feb 24 11:48 .git/
-rw-r--r--  1 zepho zepho  260 Feb 24 11:48 .gitignore
drwx------  2 zepho zepho 4096 Feb 24 11:48 invariants/
drwx------  2 zepho zepho 4096 Feb 24 11:48 observations/
drwx------  2 zepho zepho 4096 Feb 24 11:48 output/
-rw-r--r--  1 zepho zepho 2175 Feb 24 11:48 README.md
drwx------  3 zepho zepho 4096 Feb 24 11:48 snapshots/
drwx------  2 zepho zepho 4096 Feb 24 11:48 .stave/
-rw-r--r--  1 zepho zepho  152 Feb 24 11:48 stave.lock
-rw-r--r--  1 zepho zepho  350 Feb 24 11:48 stave.yaml

 tree
.
├── cli.yaml
├── invariants
├── observations
│   ├── 2026-01-11T00:00:00Z.json
│   └── 2026-01-18T00:00:00Z.json
├── output
├── README.md
├── snapshots
│   └── raw
│       └── aws-s3
│           └── README.md
├── stave.lock
└── stave.yaml

7 directories, 7 files

../stave doctor
[PASS] version-info: stave_version=0.0.1 go_version=go1.26.0 os=linux arch=amd64 binary=/home/zepho/Downloads/bizacademy-main/stave/stave
[PASS] os-version: Ubuntu 24.04.3 LTS
[PASS] shell: /bin/bash
[PASS] workspace-writable: current directory is writable: /home/zepho/Downloads/bizacademy-main/stave/beta-test
[PASS] jq: jq available
[WARN] graphviz: dot not found; graph coverage DOT output cannot be rendered to PNG
      Fix: install graphviz (https://graphviz.org/download/)
[PASS] clipboard-tool: clipboard tool available (xclip or wl-copy)
[PASS] offline-proxy-env: no proxy environment variables set
Next workflow start: stave template --step invariant > /home/zepho/Downloads/bizacademy-main/stave/beta-test/invariants/INV.S3.PUBLIC.901.yaml
zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main/stave/beta-test$ 

sudo apt install graphviz

jq
jq - commandline JSON processor [version 1.7]

Usage:	jq [options] <jq filter> [file...]
	jq [options] --args <jq filter> [strings...]
	jq [options] --jsonargs <jq filter> [JSON_TEXTS...]

jq is a tool for processing JSON inputs, applying the given filter to
its JSON text inputs and producing the filter's results as JSON on
standard output.

The simplest filter is ., which copies jq's input to its output
unmodified except for formatting. For more advanced filters see
the jq(1) manpage ("man jq") and/or https://jqlang.github.io/jq/.

Example:

	$ echo '{"foo": 0}' | jq .
	{
	  "foo": 0
	}

For listing the command options, use jq --help.

../stave doctor
[PASS] version-info: stave_version=0.0.1 go_version=go1.26.0 os=linux arch=amd64 binary=/home/zepho/Downloads/bizacademy-main/stave/stave
[PASS] os-version: Ubuntu 24.04.3 LTS
[PASS] shell: /bin/bash
[PASS] workspace-writable: current directory is writable: /home/zepho/Downloads/bizacademy-main/stave/beta-test
[PASS] jq: jq available
[PASS] graphviz: graphviz (dot) available
[PASS] clipboard-tool: clipboard tool available (xclip or wl-copy)
[PASS] offline-proxy-env: no proxy environment variables set
Next workflow start: stave template --step invariant > /home/zepho/Downloads/bizacademy-main/stave/beta-test/invariants/INV.S3.PUBLIC.901.yaml
zepho@ThinkPad-T470s-W10DG:~/Downloads/bizacademy-main/stave/beta-test$ 

 ../stave template --step invariant > /home/zepho/Downloads/bizacademy-main/stave/beta-test/invariants/INV.S3.PUBLIC.901.yaml
Next workflow start: 

stave validate --invariants /home/zepho/Downloads/bizacademy-main/stave/beta-test/invariants --observations /home/zepho/Downloads/bizacademy-main/stave/beta-test/observations  
stave evaluate --invariants /home/zepho/Downloads/bizacademy-main/stave/beta-test/invariants --observations /home/zepho/Downloads/bizacademy-main/stave/beta-test/observations --format json > /home/zepho/Downloads/bizacademy-main/stave/beta-test/output/evaluation.json

../stave validate --invariants /home/zepho/Downloads/bizacademy-main/stave/beta-test/invariants --observations /home/zepho/Downloads/bizacademy-main/stave/beta-test/observations  
WARN: Uncommitted changes detected in validate inputs (invariants/, stave.yaml). This run may not reflect committed state.
Validation passed (2 warnings)
[WARN] INVARIANT_BAD_ID_FORMAT
  error=[REDACTED]
  invariant_id=INV.S3.PUBLIC.901
  Fix: Use format INV.<DOMAIN>.<CATEGORY>.<SEQ> (e.g., INV.EXP.PUBLIC.001)

[WARN] INVARIANT_NEVER_MATCHES
  invariant_id=INV.S3.PUBLIC.901
  Fix: This may be expected if all resources are safe, or check predicate rules

---
Checked: 1 invariants, 2 snapshots, 2 resource observations

 ../stave evaluate --invariants /home/zepho/Downloads/bizacademy-main/stave/beta-test/invariants --observations /home/zepho/Downloads/bizacademy-main/stave/beta-test/observations --format json > /home/zepho/Downloads/bizacademy-main/stave/beta-test/output/evaluation.json
Resolution: context_name=beta-test invariants=/home/zepho/Downloads/bizacademy-main/stave/beta-test/invariants observations=/home/zepho/Downloads/bizacademy-main/stave/beta-test/observations project_config=/home/zepho/Downloads/bizacademy-main/stave/beta-test/stave.yaml project_config_hash=889c13449ffe4a32e4d95b072dc00ee00d966507375bfbc4b2c38ccbcb7c4523 user_config=- lockfile=/home/zepho/Downloads/bizacademy-main/stave/beta-test/stave.lock lockfile_hash=9d5cc2b432688c90b7cc80afa399cb7df58c31ec1b807b470f1560e8ee1ee180
WARN: Uncommitted changes detected in evaluate inputs (invariants/, stave.yaml). This run may not reflect committed state.
[ERR] Command failed (INTERNAL_ERROR)
Description: cannot combine explicit --invariants with enabled_invariant_packs; remove one source to keep selection deterministic
Fix: Check the command arguments and rerun with -v or -vv for additional context.
More info: https://github.com/sufield/stave/blob/main/docs/user-docs.md
