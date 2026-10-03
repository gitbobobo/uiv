import { useState } from "react";
import Library from "./Library";
import Login from "./Login";

const TOKEN_KEY = "uiv.token";

export default function App() {
  const [token, setToken] = useState(() => localStorage.getItem(TOKEN_KEY));
  const [notice, setNotice] = useState("");

  function signIn(t: string) {
    localStorage.setItem(TOKEN_KEY, t);
    setNotice("");
    setToken(t);
  }

  function signOut(reason = "") {
    localStorage.removeItem(TOKEN_KEY);
    setNotice(reason);
    setToken(null);
  }

  if (!token) return <Login notice={notice} onSignIn={signIn} />;
  return <Library token={token} onSignOut={signOut} />;
}
