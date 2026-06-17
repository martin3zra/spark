import { useForm } from "@inertiajs/react";

export default function Login({ translations = {}, errors = {} }) {
  const { data, setData, post, processing } = useForm({ email: "", password: "" });

  function submit(e) {
    e.preventDefault();
    post("/login");
  }

  return (
    <form onSubmit={submit} style={{ maxWidth: 320, margin: "4rem auto" }}>
      <h1>{translations["auth.login"] ?? "Log in"}</h1>
      <label>
        {translations["auth.email"] ?? "Email"}
        <input
          type="email"
          value={data.email}
          onChange={(e) => setData("email", e.target.value)}
        />
      </label>
      {errors?.email && <p style={{ color: "red" }}>{errors.email}</p>}
      <label>
        {translations["auth.password"] ?? "Password"}
        <input
          type="password"
          value={data.password}
          onChange={(e) => setData("password", e.target.value)}
        />
      </label>
      <button type="submit" disabled={processing}>
        {translations["auth.login"] ?? "Log in"}
      </button>
    </form>
  );
}
