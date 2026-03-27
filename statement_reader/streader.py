import pdfplumber
import argparse
import pprint
import pandas

"""
    fecha1 = cropped_page.search("01/MAR", return_groups=False, return_chars=False)
    fecha2 = cropped_page.search("02/MAR", return_groups=False, return_chars=False)
    desc = cropped_page.search("DESCRIPCION", return_groups=False, return_chars=False)
    ref = cropped_page.search("REFERENCIA", return_groups=False, return_chars=False)
    car = cropped_page.search("CARGOS", return_groups=False, return_chars=False)
    abo = cropped_page.search("ABONOS", return_groups=False, return_chars=False)
    op = cropped_page.search("OPERACION", return_groups=False, return_chars=False)
    liq = cropped_page.search("LIQUIDACION", return_groups=False, return_chars=False)
    pprint.pprint(fecha1)
    pprint.pprint(fecha2)
    pprint.pprint(desc)
    pprint.pprint(ref)
    pprint.pprint(car)
    pprint.pprint(abo)
    pprint.pprint(op)
    pprint.pprint(liq)
"""

TABLE_SETTINGS = {
    "vertical_strategy": "explicit",
    "horizontal_strategy": "text",
    "explicit_vertical_lines": [14, 59, 104, 321, 372, 416, 460, 513, 593]
}

def parse_pdf_page(page: pdfplumber.Page, top_string, bot_string):

    # Get top limit and bottom limit
    is_last = False
    top_res = page.search(top_string, return_groups=False, return_chars=False)
    bot_res = page.search(bot_string, return_groups=False, return_chars=False)
    end_transac = page.search("Total de Movimientos", return_groups=False, return_chars=False)
    
    if len(end_transac) > 0:
        bot_res = end_transac
        is_last = True

    # If can't find top or bottom exit
    if len(top_res) <= 0 or len(bot_res) <= 0:
        print("Couldn't find correct page landmarks to crop page")
        sys.exit(1)

    top = top_res[0]['bottom']
    bottom = bot_res[0]['top']
    cropped_page = page.within_bbox((0, top, page.width, bottom))
    transactions = cropped_page.extract_table(TABLE_SETTINGS)
    return transactions, is_last

def extract_transactions(args):
    trans_table = []
    with pdfplumber.open(args.filename, password=args.p__password) as pdf:
        is_last = False
        for i, page in enumerate(pdf.pages):
            tmp_table = []
            if i == 0:
                tmp_table, is_last = parse_pdf_page(
                    page,
                    "Detalle de Movimientos Realizados",
                    "La GAT Real es el rendimiento"
                )
            else:
                tmp_table, is_last = parse_pdf_page(
                    page,
                    "No. de Cliente",
                    "BBVA MEXICO, S.A., INSTITUCION DE BANCA MULTIPLE"
                )

            trans_table.extend(tmp_table)
            if is_last:
                break
    return trans_table
        

parser = argparse.ArgumentParser(
    prog='bbva-extractor',
    description='Simple python script to extract transaction information from bbva bank statements'
)

parser.add_argument('filename', type=str)
parser.add_argument('-p' '--password', type=str)
args = parser.parse_args()

if len(args.filename) <= 0:
    print("Please provide a filename")
    sys.exit(1)

table_py = extract_transactions(args)
table_df = pandas.DataFrame(table_py, columns=table_py[1])
clean_table = table_df[table_df["OPER"].str.contains("[0-9][0-9]")]
pprint.pprint(clean_table)