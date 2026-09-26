# Inbox

Raw incoming material before classification.

Examples:

- pasted text;
- voice transcript;
- URL;
- file reference;
- idea;
- note fragment;
- ambiguous instruction.

Inbox items do not need structure. The Manager later decides whether each item
becomes Work, a Note, a Project update, an Archive entry, or something else.

See `_docs/MANAGER.md` for where Inbox items and their refs live.

Inbox items stay in `Inbox/items/`. Promotion to Work updates the Inbox
frontmatter and creates a linked Work item; it does not move the Inbox file.

Raw captures use short refs like `INBOX-1`. Promoted Work tasks get separate
refs like `CORE-10`.
