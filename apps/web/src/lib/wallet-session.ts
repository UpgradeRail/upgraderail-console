import { api } from "@/lib/api";
import { signWalletMessage, type WalletNetwork } from "@/lib/wallet";

type Challenge = { id: string; message: string; nonce: string };
type Session = { address: string; network: string };

export async function authenticateWallet(address: string, network: WalletNetwork): Promise<Session> {
  const challenge = await api<Challenge>("/api/v1/auth/challenge", {
    method: "POST",
    body: JSON.stringify({ address, network: network.id, purpose: "console_session" }),
  });

  const signature = await signWalletMessage(challenge.message, network.passphrase, address);
  const verified = await api<Session>("/api/v1/auth/verify", {
    method: "POST",
    body: JSON.stringify({ challengeID: challenge.id, nonce: challenge.nonce, signature }),
  });

  if (verified.address !== address || verified.network !== network.id) {
    throw new Error("The API authenticated a different wallet or network.");
  }

  const session = await api<Session>("/api/v1/auth/session");
  if (session.address !== address || session.network !== network.id) {
    throw new Error("The authenticated session does not match the connected wallet.");
  }
  return session;
}
