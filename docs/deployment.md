# Deployment

The Next.js app can use its standalone build output. API, indexer, and worker are long-running Go services and need persistent PostgreSQL access; the indexer is not a request-time serverless function.

Required server variables are in `.env.example`. `NEXT_PUBLIC_*` values are browser-visible and must never contain database URLs, session secrets, or private RPC credentials.

Before deployment, configure an absolute `ARTIFACT_LOCAL_DIR` or an S3-compatible artifact implementation, `UPGRADERAIL_ENGINE_BIN`, PostgreSQL, Stellar RPC, controller ID, `SESSION_SECRET`, and `AUTH_DOMAIN`. No production URLs or infrastructure have been configured in this checkout.
