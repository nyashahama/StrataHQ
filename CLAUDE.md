# gstack

Use the `/browse` skill from gstack for all web browsing. Never use `mcp__claude-in-chrome__*` tools.

## Available skills

- `/browse` — browser automation via Playwright
- `/plan-ceo-review` — CEO-level plan review
- `/plan-eng-review` — engineering plan review
- `/review` — code review
- `/ship` — ship changes
- `/qa` — quality assurance
- `/setup-browser-cookies` — set up browser cookies
- `/retro` — retrospective

## Troubleshooting

If gstack skills aren't working, run the following to build the binary and register skills:

```sh
cd .claude/skills/gstack && ./setup
```
