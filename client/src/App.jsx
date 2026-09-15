import { useEffect, useState } from "react";

// Baked at build time: vite envPrefix exposes GREETING_TAG as a literal here,
// and esbuild folds this template into one string in the bundle.
const frontendLine = `frontend: hello world oxzoo-go-react_${import.meta.env.GREETING_TAG}`;

export default function App() {
  const [backend, setBackend] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    fetch("/api/greeting")
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.text();
      })
      .then((text) => setBackend(text))
      .catch((err) => setError(err.message));
  }, []);

  return (
    <main style={{ fontFamily: "system-ui, sans-serif", maxWidth: "640px", margin: "4rem auto", padding: "0 1rem" }}>
      <h1>oxzoo-go-react</h1>
      <p>{frontendLine}</p>
      <p>backend: {error ? `error: ${error}` : backend || "loading"}</p>
    </main>
  );
}
