import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Mail, Plug, PowerOff, RefreshCw } from "lucide-react";
import { useEffect, useState } from "react";

import { EmptyState } from "@/components/common/EmptyState";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { formatRelativeDate } from "@/lib/format";
import {
  emailService,
  gmailFailureMessage,
  gmailFailureOf,
} from "@/services/emailService";
import type { ApplicationStatus, DetectedEmail } from "@/types";

const STATUS_VALUES: ApplicationStatus[] = [
  "applied",
  "under-review",
  "shortlisted",
  "interview",
  "rejected",
  "hired",
];

function asStatus(value: string): ApplicationStatus | null {
  return STATUS_VALUES.includes(value as ApplicationStatus)
    ? (value as ApplicationStatus)
    : null;
}

/** Result of the Google redirect back to the dashboard, reported once per arrival. */
export interface OAuthOutcome {
  state: "connected" | "error";
  message?: string | undefined;
}

export function GmailCard({ outcome }: { outcome?: OAuthOutcome | undefined }) {
  const queryClient = useQueryClient();
  const [actionError, setActionError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    if (!outcome) return;
    setActionError(outcome.state === "error" ? outcome.message ?? null : null);
    setNotice(outcome.state === "connected" ? "Gmail connected." : null);
  }, [outcome]);

  const status = useQuery({
    queryKey: ["gmail", "status"],
    queryFn: () => emailService.getStatus(),
  });

  const connected = status.data?.connected === true;

  const emails = useQuery({
    queryKey: ["gmail", "messages"],
    queryFn: () => emailService.listEmails(),
    enabled: connected,
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["gmail"] });
  };

  const sync = useMutation({
    mutationFn: () => emailService.sync(),
    onSuccess: (result) => {
      setActionError(null);
      setNotice(
        result.processed > 0
          ? `${result.processed} new recruitment email${result.processed === 1 ? "" : "s"} imported.`
          : "No new recruitment emails found.",
      );
      invalidate();
    },
    onError: (error) => {
      setNotice(null);
      setActionError(gmailFailureMessage(error));
      const failure = gmailFailureOf(error);
      // A dead or missing grant is a state change the badge must reflect; a server
      // configuration problem or a transient Google error is not.
      if (failure === "not_connected" || failure === "reconnect_required") invalidate();
    },
  });

  const connect = useMutation({
    mutationFn: async () => {
      const url = await emailService.getConnectUrl();
      window.location.assign(url);
      return url;
    },
    onError: (error) => {
      setNotice(null);
      setActionError(gmailFailureMessage(error));
    },
  });

  const disconnect = useMutation({
    mutationFn: () => emailService.disconnect(),
    onSuccess: () => {
      setActionError(null);
      setNotice("Gmail disconnected.");
      invalidate();
    },
    onError: (error) => {
      setNotice(null);
      setActionError(gmailFailureMessage(error));
    },
  });

  const header = (
    <CardHeader className="flex-row items-center justify-between gap-3 space-y-0">
      <CardTitle className="flex items-center gap-2 font-display text-base">
        <Mail className="h-4 w-4 text-muted-foreground" aria-hidden />
        Gmail
        {status.data ? (
          <span
            className={
              connected
                ? "ml-1 rounded-full bg-success/15 px-2 py-0.5 text-xs font-medium text-success"
                : "ml-1 rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground"
            }
          >
            {connected ? "Connected" : status.data.needsReconnect ? "Needs reconnect" : "Not connected"}
          </span>
        ) : null}
      </CardTitle>
      {connected ? (
        <Button
          variant="outline"
          size="sm"
          onClick={() => sync.mutate()}
          disabled={sync.isPending}
        >
          <RefreshCw className={sync.isPending ? "h-4 w-4 animate-spin" : "h-4 w-4"} aria-hidden />
          {sync.isPending ? "Checking…" : "Sync now"}
        </Button>
      ) : null}
    </CardHeader>
  );

  if (status.isLoading) {
    return (
      <Card>
        {header}
        <CardContent className="space-y-3">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </CardContent>
      </Card>
    );
  }

  if (status.error) {
    return (
      <Card>
        {header}
        <CardContent>
          <div className="flex flex-col items-start gap-3 rounded-xl border border-destructive/30 bg-destructive/5 p-4">
            <p className="flex items-center gap-2 text-sm font-medium">
              <AlertTriangle className="h-4 w-4 text-destructive" aria-hidden />
              Gmail status unavailable
            </p>
            <p className="text-sm text-muted-foreground">
              {(status.error as Error)?.message ??
                "The XHire API did not respond, so Gmail state is unknown."}
            </p>
            <Button variant="outline" size="sm" onClick={() => status.refetch()}>
              Try again
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  const needsReconnect = status.data?.needsReconnect === true;

  return (
    <Card>
      {header}
      <CardContent className="space-y-4">
        {notice ? <p className="text-sm text-muted-foreground">{notice}</p> : null}
        {actionError ? (
          <p className="flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
            {actionError}
          </p>
        ) : null}

        {!connected ? (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {needsReconnect
                ? status.data?.reason ??
                  "Your Gmail authorization no longer works, so no emails are being imported."
                : status.data?.reason ??
                  "Connect Gmail so XHire can watch your inbox for recruitment updates."}
            </p>
            <Button onClick={() => connect.mutate()} disabled={connect.isPending}>
              <Plug className="h-4 w-4" aria-hidden />
              {connect.isPending
                ? "Opening Google…"
                : needsReconnect
                  ? "Reconnect Gmail"
                  : "Connect Gmail"}
            </Button>
          </div>
        ) : emails.isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ) : emails.error ? (
          <div className="flex flex-col items-start gap-3 rounded-xl border border-destructive/30 bg-destructive/5 p-4">
            <p className="text-sm font-medium">Imported emails could not be loaded</p>
            <p className="text-sm text-muted-foreground">
              {(emails.error as Error)?.message ?? "The XHire API did not respond."}
            </p>
            <Button variant="outline" size="sm" onClick={() => emails.refetch()}>
              Try again
            </Button>
          </div>
        ) : (emails.data ?? []).length === 0 ? (
          <EmptyState
            icon={Mail}
            title="No recruitment emails yet"
            description="Gmail is connected. Unread replies from the last two days appear here once XHire syncs your inbox."
          />
        ) : (
          <ul className="divide-y">
            {(emails.data ?? []).map((email) => (
              <EmailRow key={email.id} email={email} />
            ))}
          </ul>
        )}

        {connected ? (
          <div className="flex items-center justify-between gap-3 pt-1">
            <p className="text-xs text-muted-foreground">
              Only unread mail from the last two days is imported, and only for this account.
            </p>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => disconnect.mutate()}
              disabled={disconnect.isPending}
            >
              <PowerOff className="h-4 w-4" aria-hidden />
              Disconnect
            </Button>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function EmailRow({ email }: { email: DetectedEmail }) {
  const status = asStatus(email.suggestedStatus);

  return (
    <li className="flex items-center justify-between gap-3 py-3">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium">
          {email.detectedRole || "Role not detected"} · {email.detectedCompany || "Company not detected"}
        </p>
        <p className="truncate text-xs text-muted-foreground">
          {formatRelativeDate(email.detectedAt)}
          {email.confidence > 0 ? ` · ${email.confidence}% confidence` : ""}
          {email.classificationSource === "heuristic" ? " · subject guess" : ""}
          {email.confirmed ? " · already handled" : ""}
        </p>
      </div>
      {status ? <StatusBadge status={status} /> : null}
    </li>
  );
}
