const db = process.env.DATABASE_URL;
const stripe = process.env["STRIPE_SECRET_KEY"];
const vite = import.meta.env.VITE_API_URL;
const deno = Deno.env.get("DENO_REGION");
const node = process.env.NODE_ENV;
