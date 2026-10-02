export type EngineStatus = "BLOCKED" | "READY_WITH_WARNINGS" | "READY";

export function isBlockingStatus(status: EngineStatus): boolean {
  return status === "BLOCKED";
}
