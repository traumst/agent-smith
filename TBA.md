## mark unavailable models as such on UI

model status - available, unavailable, exhausted, - should be reflected on the UI.
if model is not available we should still see it in the list, but not be able to select it.

## add models and tools table to the settings

we should have add new section in setting with a table that displays all models supported by the API.
it should have all 4 columns from .available and .unavailable files and an additional column for status - available, unavailable, and exhausted.
we should be able to drag rows to reorder them and this order should be preserved in a dedicated file '.modelorder'.
we should be able to select default model from this table.
we should be able to refresh specific or all models status on demand.
this refresh should still be limited to once every 5 minutes on the back end. return cached status in that 5 minute window if requested.

## auto-swap to available models

if model is exhausted, we should automatically swap to the next available model in the .available list.
once exhausted it should be moved from .available to .unavailable list.
then, we should change the active model to the next preference according to '.modelorder'.
after we exhaust the list of preferred models, we should just proceed trying next arbitrary model to proceed.
if all models are exhausted - we should display that in the model selection dropdown.

## delete message

allow message deletion in chat.
it should appear as an icon similar to edit.
we should ask to confirm the deletion like we do for chats.

## handle consecutive messages from user or LLM

if we have 2 or more consecutive messages from either user or the LLM, we should merge them into a single message, when passed to the model.

## stop current working/thinking

while agent is responding, "send" button should be "stop" instead.
clicking "stop" should allow user to stop current request from proceeding.
ensure both client and backend handle this gracefully.

## token burn stats

add footer to display lifetime / monthly / weekly / daily / hourly token burn.
we should collect this data and display that on UI.
we already count how much token each request and response cost aproximately.
we should start an appent-only table that stores lifetime token spend as a series of entries.
one entry per request + response.
each entry should mention:

- iso8601 timestamp
- model used
- input token count
- output token count
- total token count
