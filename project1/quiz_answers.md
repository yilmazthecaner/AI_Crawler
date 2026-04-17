# Quiz Answers

This file answers the quiz using the current project state and the generated crawl data in `data/storage/p.data`.

## 1. Raw storage file

`data/storage/p.data`

## 2. Word that appears on multiple URLs

`python`

## 3. Three entries copied from the file

Entry 1:

```text
python http://localhost:3600/fixture/python-basics http://localhost:3600/fixture/start 1 8
```

Entry 2:

```text
python http://localhost:3600/fixture/page-signals http://localhost:3600/fixture/start 1 7
```

Entry 3:

```text
python http://localhost:3600/fixture/start http://localhost:3600/fixture/start 0 6
```

## 4. Search via the API

```text
GET http://localhost:3600/search?query=python&sortBy=relevance
```

## 5. API #1 result

- URL: `http://localhost:3600/fixture/python-basics`
- `relevance_score`: `1075`

## 6. Manual score calculation

- Entry 1 score: `( 8 x 10 ) + 1000 - ( 1 x 5 ) = 1075`
- Entry 2 score: `( 7 x 10 ) + 1000 - ( 1 x 5 ) = 1065`
- Entry 3 score: `( 6 x 10 ) + 1000 - ( 0 x 5 ) = 1060`

## 7. Does the highest calculated score match the API's #1 result?

`Yes`

## 8. How could the process be enhanced in a Chain-of-Thought manner?

The process can be improved by making each stage explicit and inspectable: fetch the raw candidate rows, normalize the query terms, score each candidate deterministically, then attach a compact explanation showing why a result ranked where it did. If the system also stored matched terms, frequency contribution, depth penalty, and tie-break metadata with each result, search responses could explain ranking step by step without requiring manual recalculation from the raw storage file.
