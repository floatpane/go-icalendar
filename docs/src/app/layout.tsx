import type { Metadata } from "next";
import "./globals.css";
import "highlight.js/styles/github-dark.css";

export const metadata: Metadata = {
	title: "go-icalendar",
	description:
		"Ergonomic iCalendar (RFC 5545) and iMIP (RFC 6047) for Go. Parse VEVENTs, build REQUEST/REPLY/CANCEL invites, expand RRULE recurrences, and compute free/busy.",
};

export default function RootLayout({
	children,
}: {
	children: React.ReactNode;
}) {
	return (
		<html lang="en">
			<body>
				<header className="site-header">
					<a href="/" className="brand">
						go-icalendar
					</a>
					<nav>
						<a href="https://github.com/floatpane/go-icalendar">GitHub</a>
					</nav>
				</header>
				<main>{children}</main>
			</body>
		</html>
	);
}

export const viewport = { width: "device-width", initialScale: 1 };
