Make two changes:
1. Add a 50-character name limit to ValidateUser (reject names longer than 50 chars)
2. In FormatUser, rename the parameter from "name" to "displayName" (pure rename, no behavior change)

After your changes, deal with any drift. `drift todo` must report "No changes detected" before you're done.
