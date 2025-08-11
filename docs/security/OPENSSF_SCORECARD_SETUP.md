# OpenSSF Scorecard Setup for Ephemos

This document explains how to complete the OpenSSF Scorecard setup for the Ephemos project.

## 🎯 What's Already Done

✅ **Scorecard workflow** - `.github/workflows/scorecard.yml` is configured  
✅ **Security policy** - `SECURITY.md` file created  
✅ **Pinned dependencies** - GitHub Actions pinned to specific SHAs  
✅ **Badge added** - README includes OpenSSF Scorecard badge (needs repo URL update)

## 🔧 Required Manual Steps

### Step 1: Create Personal Access Token (PAT)

The Scorecard workflow needs a token to access repository information for accurate scoring:

1. **Go to GitHub Settings**:
   - Navigate to https://github.com/settings/tokens?type=beta
   - Or use classic tokens: https://github.com/settings/tokens

2. **Create Fine-Grained PAT** (Recommended):
   - Click "Generate new token" → "Fine-grained personal access token"
   - **Token name**: `ephemos-scorecard-token`
   - **Expiration**: 1 year (or no expiration)
   - **Repository access**: Select "Selected repositories" → Choose your Ephemos repo
   
3. **Required Permissions** (Read-only):
   ```
   Repository permissions:
   ✅ Actions: Read
   ✅ Contents: Read  
   ✅ Metadata: Read
   ✅ Pull requests: Read
   ✅ Security events: Read
   ✅ Administration: Read (for branch protection checks)
   
   Account permissions:
   ✅ Email addresses: Read (optional)
   ```

4. **Copy the token** value (you won't see it again!)

### Step 2: Add Token to Repository Secrets

1. **Go to your Ephemos repository**
2. **Navigate to**: Settings → Secrets and variables → Actions
3. **Click**: "New repository secret"
4. **Name**: `SCORECARD_TOKEN`
5. **Value**: Paste the PAT token
6. **Click**: "Add secret"

### Step 3: Update README Badge

Replace the placeholder in README.md:

**Current (needs fixing):**
```markdown
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/your-username/ephemos/badge)](https://securityscorecards.dev/viewer/?uri=github.com/your-username/ephemos)
```

**Replace with** (use your actual GitHub username):
```markdown
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/YOUR_USERNAME/ephemos/badge)](https://securityscorecards.dev/viewer/?uri=github.com/YOUR_USERNAME/ephemos)
```

### Step 4: Enable Branch Protection (For Higher Score)

1. **Go to**: Settings → Branches
2. **Click**: "Add branch protection rule"
3. **Branch name pattern**: `main`
4. **Enable these settings**:
   ```
   ✅ Require a pull request before merging
   ✅ Require approvals (at least 1)
   ✅ Require status checks to pass before merging
   ✅ Require branches to be up to date before merging
   ✅ Require conversation resolution before merging
   ✅ Restrict pushes that create files
   ✅ Do not allow bypassing the above settings
   ```

### Step 5: Trigger Initial Scorecard Run

1. **Push any commit** to the main branch (or run workflow manually)
2. **Check Actions tab** for "Scorecard supply-chain security" workflow
3. **Wait for completion** (usually 2-3 minutes)
4. **Check Security tab** → Code scanning for detailed results

## 📊 Expected Scorecard Results

After setup, Ephemos should score well because it already has:

### Strong Security Practices ✅
- **CI-Tests**: Comprehensive test suite
- **SAST**: CodeQL and Semgrep scanning
- **Vulnerabilities**: govulncheck, OSV Scanner, Grype
- **Dependency-Update-Tool**: Automated dependency PRs
- **Security-Policy**: Comprehensive SECURITY.md
- **Pinned-Dependencies**: GitHub Actions pinned to SHAs
- **SBOM**: Software Bill of Materials generation

### Areas for Improvement ⚠️
- **Signed-Releases**: Not yet implemented (can add later)
- **Branch-Protection**: Depends on repository settings (Step 4 above)
- **Code-Review**: Depends on using PR workflow consistently

## 🎯 Expected Score Range

**Before branch protection**: 7-8/10  
**After branch protection**: 8-9/10  
**After signed releases**: 9-10/10

## 🔍 Monitoring and Maintenance

### Weekly Updates
- Scorecard runs automatically every Saturday at 1:30 AM UTC
- Results are published to scorecard.dev
- Badge updates automatically

### Viewing Results
- **Badge**: Shows current score in README
- **Detailed view**: Click badge → opens scorecard.dev results
- **GitHub Security**: Security tab → Code scanning → Scorecard alerts

### Improving Score
1. **Check scorecard.dev** for specific recommendations
2. **Review Security alerts** in GitHub
3. **Fix issues** and push changes
4. **Monitor score improvement** on next run

## 🚨 Troubleshooting

### Workflow Fails
- **Check token permissions**: Ensure SCORECARD_TOKEN has correct scopes
- **Verify repository is public**: Private repos need GitHub Advanced Security
- **Check workflow logs**: Actions tab → Failed run → View logs

### Badge Shows "unknown"
- **Wait 24-48 hours** after first successful run
- **Check workflow success**: Must run successfully at least once
- **Verify publish_results**: Must be set to `true` (already configured)

### Low Score Initially
- **Normal behavior**: Most projects start with 4-6/10
- **Gradual improvement**: Address alerts one by one
- **Focus on high-impact**: Branch protection gives biggest boost

## 📚 Resources

- **OpenSSF Scorecard**: https://github.com/ossf/scorecard
- **Scorecard Action**: https://github.com/ossf/scorecard-action
- **Best Practices Guide**: https://best.openssf.org/
- **Ephemos Security Docs**: See `docs/security/` directory

---

**Once you complete Steps 1-3, the OpenSSF Scorecard badge will work and show your security score!**