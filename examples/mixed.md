# Deploy Checklist :ship:

> :warning: **Warning**: Make sure CI is green before deploying!

## Steps

1. Pull latest from `main`
2. Run tests:
   ```
   go test ./...
   ```
3. Check the [changelog](https://example.com/changelog)
4. Tag the release: `git tag v2.0.0`
5. Verify the [deploy dashboard](https://example.com/deploy)

---

## Contacts

| Role | Name | Slack |
|------|------|-------|
| Lead | Alice | @alice |
| SRE | Bob | @bob |

## Notes

- :x: ~~Old deploy process~~ is deprecated
- :white_check_mark: Use the **new** pipeline _exclusively_
- :book: See `docs/deploy.md` for details
- :link: Reference the [runbook](https://example.com/runbook) and [incident process](https://example.com/incidents)

![architecture](https://go.dev/blog/gopher/header.jpg)
