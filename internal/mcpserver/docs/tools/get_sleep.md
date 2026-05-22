Return sleep data for a date or cycle ID.

Sleep events can cross midnight and should be interpreted through WHOOP cycle
semantics. If a date is provided, thoop resolves the cycle first, then loads the
sleep associated with that cycle.
