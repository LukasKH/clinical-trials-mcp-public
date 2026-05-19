# Coding Agent Guide

This repo is only `clinical-trials-mcp`: a Go Streamable HTTP MCP server plus
Cloud Run Terraform and GitHub Actions deployment.

## Commands

- Run locally: `cd apps/clinical-trials-mcp && go run .`
- Test: `cd apps/clinical-trials-mcp && go test -race ./...`
- Build image: `docker build -f apps/clinical-trials-mcp/Dockerfile -t clinical-trials-mcp .`
- Validate Terraform: `terraform -chdir=infra/remote-mcp fmt -check -recursive && terraform -chdir=infra/remote-mcp init -backend=false && terraform -chdir=infra/remote-mcp validate`

## Guardrails

- Keep registry API mapping in `apps/clinical-trials-mcp/clinicaltrials/`.
- Preserve MCP tool names and parameter shapes unless the task changes them.
- Do not run `terraform apply`, `gcloud run deploy`, or other production-changing commands without explicit approval.
