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

---

bash examples/demo-s3-public-read/run.sh

════════════════════════════════════════════
  Findings — Public Read via Bucket Policy
════════════════════════════════════════════

{
  "summary": {
    "total_assets": 1,
    "exposed_resources": 1,
    "violations": 35
  },
  "status": "NON_COMPLIANT",
  "controls": [],
  "findings_count": 0
}

════════════════════════════════════
  Encoding — fact projection check
════════════════════════════════════

Encoding verified: 3/3 verifiable facts match observations ✓


Why is controls array empty?
Why is findings count 0 but violations 35?

---

ave$ bash examples/demo-ai-security/run.sh
ACT 1: What does Stave find?

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Running: stave apply on a Bedrock agent + Lambda + KB + S3 PHI bucket

AI findings: 0 critical, 0 high, 0 medium, 0 low — 0 total


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ACT 2: Those aren't separate findings. They're attack chains.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Compound chains: 11

  [CRITICAL] bedrock_agent_overpermissioned
      Bedrock agent execution role grants broad lambda:InvokeFunction, the agent has no associated guardrail filtering prompts and outputs, and per-agent invocation logging is disabled. An attacker who gains prompt control can invoke arbitrary Lambda functions through the agent with no content filtering and no audit trail. Threshold of 3 because all three legs of the compound — broad reach, no filter, no detection — must stack to make the agent a usable exfiltration / lateral-movement primitive. Drop any one and the attacker has either limited reach, blocked content, or visible activity.


  [CRITICAL] bedrock_agent_tool_phi_exposure
      A Lambda function is registered as a Bedrock agent action group AND that Lambda's execution role can read S3 objects from a PHI-tagged bucket. The agent's prompt-driven tool invocation reaches PHI through the Lambda → S3 path. Compose-by-asset.ID on the Lambda — both the bedrock-tool marker and the role-reaches-PHI violation fire on the same Lambda asset, so no scope_field is required. The agent → Lambda hop is established by the marker (Lambda is registered in some agent's actionGroups); the Lambda → PHI hop is established by the violation (Lambda role's effective S3 set intersects PHI buckets). Threshold of 2 because both legs must fire — a Lambda reaching PHI without being an agent tool is a different control's job; an agent tool that doesn't reach PHI is the desired state.


  [CRITICAL] bedrock_rag_phi_exposure
      Bedrock knowledge base indexes an S3 bucket AND that bucket carries the data-classification=phi marker. RAG retrieval through the agent connected to this knowledge base returns PHI to whoever queries the agent. Same compound shape as cognito_unauth_phi_s3 but on the AI-data axis: each side is a fact-recording marker (KB indexes bucket X; bucket X is PHI), neither a violation alone — the dangerous combination is the join on a shared bucket ARN. The Bedrock side stamps target_bucket_arn on the knowledge base; the S3 side's scope is its own ARN (asset.ID), which the resolver returns when the property path is absent on the storage asset.


  [CRITICAL] iam_boundary_governance_failure
      ANY boundary failure — self-modifiable, missing on delegated roles, or wildcard actions. Each independently defeats the permission boundary model. Threshold 1 because any single failure makes boundaries ineffective as a security control.


  [CRITICAL] iam_boundary_governance_failure
      ANY boundary failure — self-modifiable, missing on delegated roles, or wildcard actions. Each independently defeats the permission boundary model. Threshold 1 because any single failure makes boundaries ineffective as a security control.


  [CRITICAL] iam_boundary_governance_failure
      ANY boundary failure — self-modifiable, missing on delegated roles, or wildcard actions. Each independently defeats the permission boundary model. Threshold 1 because any single failure makes boundaries ineffective as a security control.


  [CRITICAL] iam_boundary_governance_failure
      ANY boundary failure — self-modifiable, missing on delegated roles, or wildcard actions. Each independently defeats the permission boundary model. Threshold 1 because any single failure makes boundaries ineffective as a security control.


  [CRITICAL] iam_escalation_undetected
      No PassRole alarm plus either no trust policy change alarm or no cross-account assumption alarm. The two most important escalation detection points — PassRole and trust modification — are unmonitored. An attacker escalates privileges without triggering any real-time alert.


  [CRITICAL] iam_escalation_undetected
      No PassRole alarm plus either no trust policy change alarm or no cross-account assumption alarm. The two most important escalation detection points — PassRole and trust modification — are unmonitored. An attacker escalates privileges without triggering any real-time alert.


  [CRITICAL] iam_escalation_undetected
      No PassRole alarm plus either no trust policy change alarm or no cross-account assumption alarm. The two most important escalation detection points — PassRole and trust modification — are unmonitored. An attacker escalates privileges without triggering any real-time alert.


  [CRITICAL] iam_escalation_undetected
      No PassRole alarm plus either no trust policy change alarm or no cross-account assumption alarm. The two most important escalation detection points — PassRole and trust modification — are unmonitored. An attacker escalates privileges without triggering any real-time alert.



━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ACT 3: Engines independently confirm the encoding.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Facts exported: 6613

Encoding verification:
  Encoding verified: 9/9 verifiable facts match observations ✓

SMT solver available:
  Z3:    Z3 version 4.8.12 - 64 bit
  cvc5:  This is cvc5 version 1.1.2

  Stave exports SMT-LIB v2 facts ready for these solvers.
  Per-control forbidden_state queries live under
  examples/z3-forbidden-state/ — for controls that define them,
  Z3 + cvc5 + Yices independently verify the unsafe state is
  reachable or impossible. The Iteration 2–6 AI controls do not
  yet declare forbidden_state (they are CEL predicates on
  pre-computed collector facts) — the SMT layer is the natural
  Iteration 7 extension.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ACT 4: The fix — and proof it works.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Remediations applied to the four assets:
  1. Scoped agent execution role to specific Lambda ARNs
  2. Attached a Bedrock guardrail (sensitive-info filter on)
  3. Enabled per-agent invocation logging
  4. Changed KB data source from PHI bucket → product-docs bucket
  5. Scoped Lambda tool role to non-PHI bucket only

Running: stave apply on the remediated configuration

AI findings: 0
Chains:      10


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ACT 5: What a component scanner reports on the same writeup.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  (Illustrative — these are the checks a component-level
   scanner would run; not actual output from any vendor.)

    Bedrock encryption:        ✅ PASS — invocation logs use KMS
    Bedrock VPC endpoint:      ✅ PASS — VPC endpoint configured
    Bedrock model allowlist:   ✅ PASS — foundation models scoped
    S3 encryption at rest:     ✅ PASS — SSE-KMS enabled
    S3 public access block:    ✅ PASS — public access blocked
    Lambda env encryption:     ✅ PASS — at-rest encryption on

  6 checks. 6 passes. Component scanner: COMPLIANT.

  Stave on the same configuration: 11 CRITICAL compound chains.
  Agent → Lambda → S3 PHI. No guardrail. No audit trail.

  The scanner checked components. Stave checked interactions.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Demo complete.

  Findings:    0 on writeup  →  0 on remediated
  AI findings: 0 on writeup  →  0 on remediated
  Chains:      11 on writeup  →  10 on remediated
  Catalog:     3164 controls across 74 AWS service domains
  Source:      github.com/sufield/stave
zepho@ThinkPad-T470s-W10DG:~/work/sufield/stave$ 

---

ADDITIONAL COMMANDS
    alias            Manage command aliases
    attest           Snapshot tamper detection via Ed25519 signatures
    bisect           Find when a control was first violated
    bundle           Generate a sealed evidence bundle for air-gap GRC integration
    capabilities     Print supported input types and version constraints (default) or a user-facing catalog (subcommand)
    cel              CEL expression tools
    check            Compare before/after evaluations to check remediation
    compare          Compare compliance posture between two frameworks
    compliance       Evaluate a snapshot against a compliance framework and report coverage
    contract         Inspect Stave's per-asset-type input contracts
    controls         Work with control definitions
    coverage         Analyze observation field coverage against control predicates
    diff             Compare two observation snapshots or control catalogs
    discover         Resolve AWS services to the data Stave needs (the collection manifest)
    doctor           Check local environment readiness for Stave workflows
    exempt           Manage risk acceptances (acknowledgments, exceptions, exemptions)
    export           Export controls and compliance evidence
    export-controls  Export the control catalog for external solver consumption
    export-sir       Export the Stave Intermediate Representation as JSON
    fingerprint      Policy fingerprint diagnostics
    fmt              Format control and observation files deterministically
    gaps             Report which observation properties are absent + what they unlock
    graph            Visualize control and asset relationships
    lint             Lint control files for design quality
    map              ATT&CK tactic coverage and gap analysis
    metrics          Write Prometheus scrape file for node_exporter
    pack             Concern packs — named control groupings and their data requirements
    packs            Inspect built-in control packs
    path             Export attack path graph data from active chain findings
    permissions      Query net effective permissions from a snapshot
    plan             Preview which controls will evaluate, by service and severity
    profile          Manage compliance profiles
    prove            Run Z3 SMT queries against a Stave assessment
    readiness        Report what Stave can/can't evaluate given the supplied observations
    recommend        Recommend templates for a snapshot
    render           Render JSON data through a Go text/template
    sanitize         Sanitize a snapshot for cross-boundary sharing
    schemas          List all contract schemas
    score            Compute security posture score (0-100)
    scorecard        Multi-framework compliance scorecard
    search           Find catalog entries matching a free-form intent
    telemetry        Emit structured NDJSON telemetry from assessment output
    template         Manage assessment templates
    test             Run embedded control test cases
    toolmap          Map offensive tools to configuration prerequisites and find coverage gaps
    transform        Convert raw AWS CLI snapshots into obs.v0.1 observations (built-in jq)
    trend            Analyze compliance posture trends across assessment runs
    validate-mapping Validate a Steampipe→Stave mapping file before use
    version          Print version and environment state

