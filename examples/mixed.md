# Deploy Checklist 🚢

> ⚠️ **Warning**: Make sure CI is green before deploying!

## Steps

1. Pull latest from `main`
2. Run tests:
   ```
   go test ./...
   ```
3. Check the [changelog](https://example.com/changelog)
4. Tag the release: `git tag v2.0.0`

---

## Contacts

| Role | Name | Slack |
|------|------|-------|
| Lead | Alice | @alice |
| SRE | Bob | @bob |

## Notes

- ❌ ~~Old deploy process~~ is deprecated
- ✅ Use the **new** pipeline _exclusively_
- 📖 See `docs/deploy.md` for details

![architecture](https://go.dev/blog/gopher/header.jpg)
