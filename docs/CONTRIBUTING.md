# Contributing to Ephemos

## Code Review Process

Ephemos follows strict security practices including mandatory code review for all changes.

### Pull Request Requirements

All code changes must:

1. **Be submitted via Pull Request** - No direct pushes to main branch
2. **Receive at least 1 approval** from a code owner
3. **Pass all required status checks** including:
   - CI/CD Pipeline (build, test, lint)
   - Security scans (CodeQL, secrets scanning)
   - SBOM generation and validation
4. **Have all conversations resolved** before merging

### Review Guidelines

#### For Contributors
- Create feature branches from latest main
- Write clear PR descriptions explaining the change
- Ensure all tests pass before requesting review
- Respond to review feedback promptly
- Keep PRs focused and reasonably sized

#### For Reviewers
- Review for security vulnerabilities
- Check code quality and maintainability  
- Verify tests cover new functionality
- Ensure documentation is updated
- Test changes locally when possible

### Security-Critical Changes

Changes to these areas require extra scrutiny:
- Authentication and authorization logic
- Configuration management
- CI/CD workflows and scripts
- Dependency updates
- Container and deployment configurations

### Emergency Procedures

For critical security fixes:
1. Create PR as normal
2. Request expedited review
3. Ensure security team validates the fix
4. Follow up with post-incident review

## Code Owners

See `.github/CODEOWNERS` for the list of code owners who can approve changes to specific areas of the codebase.

## Getting Started

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Make your changes and test thoroughly
4. Submit a pull request with clear description
5. Wait for review and address any feedback
6. Celebrate when your PR is merged! 🎉

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for questions or ideas
- Review existing PRs to understand the process