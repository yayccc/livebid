import { Search } from 'lucide-react'
import { Toast } from 'antd-mobile'

export function SearchOverlay() {
  function handleSearchClick() {
    Toast.show('搜索功能开发中')
  }

  return (
    <button type="button" className="search-overlay" onClick={handleSearchClick}>
      <Search size={16} aria-hidden="true" />
      <span>搜索直播间、主播、商品</span>
    </button>
  )
}
