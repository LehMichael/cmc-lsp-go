# Siemens Create MyConfig manual

This directory contains both English and German editions of the Siemens
"Create MyConfig - Diff, Expert, Topo" operating manual:

| Edition | English | German |
| --- | --- | --- |
| 06/2019 | `840Dsl_Create_MyConfig_op_man_en-US.pdf` (A5E36537479B-AG) | `840Dsl_Create_MyConfig_op_man_0619_de_de-DE.pdf` (A5E36537479A-AG) |
| 06/2026 | `ONE_Create_MyConfig_op_man_0626_en-US.pdf` (A5E50115644B AL) | `ONE_Create_MyConfig_op_man_0626_de-DE.pdf` (A5E50115644A AL) |

Each adjacent `.txt` file is a complete layout-preserving extraction generated
with, for example:

```sh
pdftotext -layout 840Dsl_Create_MyConfig_op_man_en-US.pdf \
  840Dsl_Create_MyConfig_op_man_en-US.txt
```

Search the text first, then inspect the matching PDF pages whenever table
columns or visual structure affect the interpretation.

The 06/2019 edition documents some legacy behavior and library syntax in more
detail. The 06/2026 edition is the primary reference for current language
features; compare its English and German editions when one contains an
ambiguous heading or signature.

Official sources:

- PDF: <https://cache.industry.siemens.com/dl/files/192/109769192/att_991335/v1/840Dsl_Create_MyConfig_op_man_en-US.pdf>
- Searchable HTML archive: <https://support.industry.siemens.com/cs/attachments/109769192/125621338379.zip?download=true>
