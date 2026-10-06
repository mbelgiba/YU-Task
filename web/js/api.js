export async function loadAppConfig() {
  const response = await fetch("/api/config", { headers: { Accept: "application/json" } });
  if (!response.ok) throw new Error(`Configuration request failed (${response.status})`);
  return response.json();
}
