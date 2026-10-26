export function Fiche({ texte, html }: { texte: string; html: string }) {
  const sur = (
    // ok: ts-dangerously-set-inner-html
    <p>{texte}</p>
  );
  const interdit = (
    // ruleid: ts-dangerously-set-inner-html
    <p dangerouslySetInnerHTML={{ __html: html }} />
  );
  return [sur, interdit];
}
