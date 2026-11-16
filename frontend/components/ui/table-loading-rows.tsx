export default function TableLoadingRows({ columns, rows = 5 }: { columns: number; rows?: number }) {
  return <>
    {Array.from({ length: rows }, (_, row) => <tr key={row} className="admin-table-skeleton-row" aria-hidden={row > 0}>
      {Array.from({ length: columns }, (_, column) => <td key={column}>
        {row === 0 && column === 0 && <span className="visually-hidden" role="status">Memuat data tabel...</span>}
        <span className="admin-table-skeleton-bar" aria-hidden="true" style={{ width: `${56 + (row * 19 + column * 13) % 39}%` }} />
      </td>)}
    </tr>)}
  </>;
}
