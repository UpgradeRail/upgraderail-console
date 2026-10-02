"use client";

import { useEffect, useState } from "react";
import { ApiError, api } from "@/lib/api";

type Resource<T> = { kind: "loading" } | { kind: "ready"; value: T[] } | { kind: "error"; message: string };

export function DataList<T extends Record<string, unknown>>({ endpoint, empty, columns }: { endpoint: string; empty: string; columns: ReadonlyArray<{ key: keyof T; label: string }> }) {
  const [resource, setResource] = useState<Resource<T>>({ kind: "loading" });
  useEffect(() => { void api<T[]>(endpoint).then((value) => setResource({ kind: "ready", value })).catch((error: unknown) => setResource({ kind: "error", message: error instanceof ApiError ? error.message : "The request could not be completed." })); }, [endpoint]);
  if (resource.kind === "loading") return <section className="empty-state"><span className="eyebrow">LOADING</span><h2>Reading indexed data</h2><p>Waiting for the Console API.</p></section>;
  if (resource.kind === "error") return <section className="empty-state"><span className="eyebrow">API UNAVAILABLE</span><h2>Data could not be loaded</h2><p>{resource.message}</p></section>;
  if (resource.value.length === 0) return <section className="empty-state"><span className="eyebrow">NO RESULTS</span><h2>{empty}</h2><p>The indexer has not produced matching records.</p></section>;
  return <div className="data-table-wrap"><table><thead><tr>{columns.map((column) => <th key={String(column.key)}>{column.label}</th>)}</tr></thead><tbody>{resource.value.map((row, index) => <tr key={String(row.id ?? index)}>{columns.map((column) => <td key={String(column.key)}>{stringify(row[column.key])}</td>)}</tr>)}</tbody></table></div>;
}

function stringify(value: unknown): string { return value === null || value === undefined ? "—" : typeof value === "string" || typeof value === "number" ? String(value) : JSON.stringify(value); }
