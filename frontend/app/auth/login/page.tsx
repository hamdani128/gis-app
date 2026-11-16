import LoginPage from "../login";

export default async function Page({ searchParams }: { searchParams: Promise<{ reason?: string }> }) {
  const params = await searchParams;
  return <LoginPage idleExpired={params.reason === "idle"} />;
}
