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
exhausted models should be moved from .available to .unavailable list.
then, we should change the active model to the next preference according to '.modelorder'.
after we exhaust the list of all available preferred models, we should try to use another arbitrary model to proceed.
if all models are exhausted - we should display that in the model selection dropdown.

## delete message

allow message deletion in chat.
it should appear as an icon similar to edit.
we should ask to confirm the deletion like we do for chats.

## handle consecutive messages from user or LLM

The models expect pre-defined structure to in converstation.
Specifically, they demand a strict user->assistant alternating sequence.
if we have 2 or more consecutive messages from either user or the LLM, we should merge them into a single message in chat history.
This merge should only be reflected in the history passed to the model, not in the UI.

## stop current working/thinking

alow users to cancel current request and drop the partial response from history.
while agent is responding, "send" button should become "stop" instead.
clicking "stop" should allow user to stop current request from proceeding.
ensure both client and backend handle this gracefully.
once model has responded fully, "stop" should turn back to "send".

## display token burn stats

we already count how much tokens each request and response cost aproximately.
we should create appent-only tables that store token spend as a series of entries.
hourly table should have two entries for request and response tokens.
each entry should mention:

- iso8601 timestamp - when record was inserted
- model used - 1-to-1 as in .available file
- input token count - tokens burned in request to LLM
- output token count - tokens burned in response from LLM
- total token count - total tokens burned in request-response cycle

we should collect lifetime / monthly / weekly / daily / hourly token usage data to inform the user about their usage.
create dedicated 'burn' table for each of these time periods: lifetime / monthly / weekly / daily / hourly.
hourly should be appended with each request/response cycle, as two entries: one for request tokens, one for response tokens.
data for longer periods should be aggregated from the shorter:
daily from houry, weekly from daily, monthly from weekly, lifetime from monthly.
we want to keep in-memory estimation and update it with actual data from DB whenever needed.

each time we aggregate data from smaller time period into larger one, we should drop the data for smaller time period from its table.

add footer on UI to display lifetime / monthly / weekly / daily / hourly token burn.

##
