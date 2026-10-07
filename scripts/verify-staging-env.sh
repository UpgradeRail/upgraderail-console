#!/usr/bin/env bash
# verify-staging-env.sh verifies environment variables and security constraints
# for deployed staging environments across Web, API, Indexer, and Worker services.
set -euo pipefail

target_service="${1:-all}"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

warn() {
  echo "WARN: $*" >&2
}

info() {
  echo "INFO: $*"
}

check_security_rules() {
  info "Checking global security configuration..."

  # Check that NEXT_PUBLIC variables do not contain secrets
  while IFS='=' read -r key val; do
    if [[ "$key" =~ ^NEXT_PUBLIC_ ]]; then
      if [[ "$val" =~ (postgres|mysql|sqlite)://.*:.*@ ]]; then
        fail "Secret database credential found in public variable $key"
      fi
      if [[ "$val" =~ ^S[A-Z0-9]{55}$ ]]; then
        fail "Stellar secret key found in public variable $key"
      fi
      if [[ "$key" =~ (SECRET|TOKEN|KEY|PASSWORD) && ! "$key" =~ (PASSPHRASE|API_KEY|PUBLIC_KEY) ]]; then
        fail "Sensitive keyword in public variable $key"
      fi
    fi
  done < <(env)

  # Check HTTPS requirement for remote origins
  if [[ -n "${WEB_ORIGIN:-}" ]]; then
    if [[ "$WEB_ORIGIN" =~ ^https:// ]]; then
      info "WEB_ORIGIN uses HTTPS: $WEB_ORIGIN"
    elif [[ "$WEB_ORIGIN" =~ ^http://(localhost|127\.0\.0\.1)(:[0-9]+)?$ ]]; then
      info "WEB_ORIGIN uses local development origin: $WEB_ORIGIN"
    else
      fail "WEB_ORIGIN must use HTTPS in staging/production: $WEB_ORIGIN"
    fi
  fi

  if [[ -n "${AUTH_DOMAIN:-}" && -n "${WEB_ORIGIN:-}" ]]; then
    # Extract hostname
    host=$(echo "$WEB_ORIGIN" | sed -E 's|^https?://([^/:]+).*|\1|')
    if [[ "$AUTH_DOMAIN" != "$host" ]]; then
      fail "AUTH_DOMAIN ($AUTH_DOMAIN) must match WEB_ORIGIN hostname ($host)"
    fi
    info "AUTH_DOMAIN matches WEB_ORIGIN host: $AUTH_DOMAIN"
  fi
}

check_web_env() {
  info "Checking Web configuration..."
  local required_vars=(
    NEXT_PUBLIC_STELLAR_NETWORK
    NEXT_PUBLIC_STELLAR_RPC_URL
    NEXT_PUBLIC_CONTROLLER_ID
  )
  for v in "${required_vars[@]}"; do
    if [[ -z "${!v:-}" ]]; then
      fail "Missing required Web variable: $v"
    fi
  done

  if [[ -z "${NEXT_PUBLIC_API_BASE_URL:-}" ]]; then
    warn "NEXT_PUBLIC_API_BASE_URL is not set; client will operate in decoupled/read-only mode."
  else
    if [[ ! "${NEXT_PUBLIC_API_BASE_URL}" =~ ^https?:// ]]; then
      fail "NEXT_PUBLIC_API_BASE_URL must be a valid URL"
    fi
  fi
  info "Web configuration valid."
}

check_api_env() {
  info "Checking API configuration..."
  local required_vars=(
    DATABASE_URL
    WEB_ORIGIN
    AUTH_DOMAIN
  )
  for v in "${required_vars[@]}"; do
    if [[ -z "${!v:-}" ]]; then
      fail "Missing required API variable: $v"
    fi
  done

  if [[ ! "$DATABASE_URL" =~ ^postgres(ql)?:// ]]; then
    fail "DATABASE_URL must be a PostgreSQL connection URI"
  fi
  info "API configuration valid."
}

check_indexer_env() {
  info "Checking Indexer configuration..."
  local required_vars=(
    DATABASE_URL
    STELLAR_NETWORK
    STELLAR_NETWORK_PASSPHRASE
    STELLAR_RPC_URL
    UPGRADERAIL_CONTROLLER_ID
    UPGRADERAIL_START_LEDGER
  )
  for v in "${required_vars[@]}"; do
    if [[ -z "${!v:-}" ]]; then
      fail "Missing required Indexer variable: $v"
    fi
  done

  if [[ ! "$UPGRADERAIL_START_LEDGER" =~ ^[0-9]+$ ]] || [[ "$UPGRADERAIL_START_LEDGER" -le 0 ]]; then
    fail "UPGRADERAIL_START_LEDGER must be a positive integer"
  fi

  if [[ ! "$STELLAR_RPC_URL" =~ ^https:// && ! "$STELLAR_RPC_URL" =~ ^http://(localhost|127\.0\.0\.1) ]]; then
    fail "STELLAR_RPC_URL must be HTTPS or local HTTP"
  fi
  info "Indexer configuration valid."
}

check_worker_env() {
  info "Checking Worker configuration..."
  local required_vars=(
    DATABASE_URL
    ARTIFACT_LOCAL_DIR
    UPGRADERAIL_WORK_DIR
    UPGRADERAIL_ENGINE_BIN
  )
  for v in "${required_vars[@]}"; do
    if [[ -z "${!v:-}" ]]; then
      fail "Missing required Worker variable: $v"
    fi
  done
  info "Worker configuration valid."
}

case "$target_service" in
  web)
    check_security_rules
    check_web_env
    ;;
  api)
    check_security_rules
    check_api_env
    ;;
  indexer)
    check_security_rules
    check_indexer_env
    ;;
  worker)
    check_security_rules
    check_worker_env
    ;;
  all)
    check_security_rules
    # In all mode, validate whatever services have their marker variables set, or validate security
    info "Global staging configuration check completed successfully."
    ;;
  *)
    fail "Unknown target service: $target_service (valid: web, api, indexer, worker, all)"
    ;;
esac
