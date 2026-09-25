/**
 * Renders model output: keeps line breaks and turns **bold** into <strong>.
 * Works on React text nodes, never raw HTML, so model output cannot inject
 * markup.
 */
export function MessageText({ text }: { text: string }) {
  const parts = text.split(/(\*\*[^*]+\*\*)/g);
  return (
    <p className="whitespace-pre-wrap break-words leading-relaxed">
      {parts.map((part, i) =>
        part.startsWith("**") && part.endsWith("**") && part.length > 4 ? (
          <strong key={i}>{part.slice(2, -2)}</strong>
        ) : (
          part
        ),
      )}
    </p>
  );
}
