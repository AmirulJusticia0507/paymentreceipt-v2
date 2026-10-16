// Package paymentreceiptstore adapts sqlc queries to payment receipt use cases.
package paymentreceiptstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/paymentreceipt"
)

type Store struct{ q *dbgen.Queries }
func New(q *dbgen.Queries)*Store{return &Store{q:q}}
func text(v string)pgtype.Text{return pgtype.Text{String:v,Valid:v!=""}}
func id(v int64)pgtype.Int8{return pgtype.Int8{Int64:v,Valid:v>0}}
func optID(v *int64)pgtype.Int8{if v==nil{return pgtype.Int8{}};return id(*v)}
func date(v *time.Time)pgtype.Date{if v==nil{return pgtype.Date{}};return pgtype.Date{Time:*v,Valid:true}}
func status(v string)dbgen.NullPaymentStatus{valid:=v=="pending"||v=="checked"||v=="processed"||v=="finished"||v=="cancelled";return dbgen.NullPaymentStatus{PaymentStatus:dbgen.PaymentStatus(v),Valid:valid}}
func stringPtr(v pgtype.Text)*string{if !v.Valid{return nil};s:=v.String;return &s}
func intPtr(v pgtype.Int8)*int64{if !v.Valid{return nil};n:=v.Int64;return &n}
func fromList(v dbgen.ListPaymentReceiptsRow)paymentreceipt.Receipt{return paymentreceipt.Receipt{ID:v.ID,Kode:v.Kode,RequestID:v.RequestID,RequestKode:v.RequestKode,RequestNama:v.RequestNama,DepartmentID:v.DepartmentID,DepartmentKode:v.DepartmentKode,DepartmentNama:v.DepartmentNama,SourceID:intPtr(v.SourceID),SourceMode:v.SourceMode,SourceNamaBarang:v.SourceNamaBarang.String,CategoryID:intPtr(v.CategoryID),CategoryNama:v.CategoryNama.String,UnitID:intPtr(v.UnitID),UnitNama:v.UnitNama.String,TanggalPembelian:v.TanggalPembelian.Time,Usage:v.Usage,DeskripsiItems:v.DeskripsiItems,JumlahItems:v.JumlahItems,HargaItems:v.Hargaitems,TotalHarga:v.TotalHarga,PriceEstimate:stringPtr(v.PriceEstimate),Keterangan:v.Keterangan,Status:string(v.Status),InputDate:v.Inputdate,CreatedAt:v.CreatedAt,UpdatedAt:v.UpdatedAt}}
func fromGet(v dbgen.GetPaymentReceiptRow)paymentreceipt.Receipt{return paymentreceipt.Receipt{ID:v.ID,Kode:v.Kode,RequestID:v.RequestID,RequestKode:v.RequestKode,RequestNama:v.RequestNama,DepartmentID:v.DepartmentID,DepartmentKode:v.DepartmentKode,DepartmentNama:v.DepartmentNama,SourceID:intPtr(v.SourceID),SourceMode:v.SourceMode,SourceNamaBarang:v.SourceNamaBarang.String,CategoryID:intPtr(v.CategoryID),CategoryNama:v.CategoryNama.String,UnitID:intPtr(v.UnitID),UnitNama:v.UnitNama.String,TanggalPembelian:v.TanggalPembelian.Time,Usage:v.Usage,DeskripsiItems:v.DeskripsiItems,JumlahItems:v.JumlahItems,HargaItems:v.Hargaitems,TotalHarga:v.TotalHarga,PriceEstimate:stringPtr(v.PriceEstimate),Keterangan:v.Keterangan,Status:string(v.Status),InputDate:v.Inputdate,CreatedAt:v.CreatedAt,UpdatedAt:v.UpdatedAt}}
func (s *Store) List(ctx context.Context,f paymentreceipt.Filters)([]paymentreceipt.Receipt,int64,error){p:=dbgen.ListPaymentReceiptsParams{Status:status(f.Status),DepartmentID:id(f.DepartmentID),RequestID:id(f.RequestID),DateFrom:date(f.DateFrom),DateTo:date(f.DateTo),Q:text(f.Query),Sort:f.Sort,Limit:f.Limit,Offset:f.Offset};rows,err:=s.q.ListPaymentReceipts(ctx,p);if err!=nil{return nil,0,err};total,err:=s.q.CountPaymentReceipts(ctx,dbgen.CountPaymentReceiptsParams{Status:p.Status,DepartmentID:p.DepartmentID,RequestID:p.RequestID,DateFrom:p.DateFrom,DateTo:p.DateTo,Q:p.Q});if err!=nil{return nil,0,err};out:=make([]paymentreceipt.Receipt,0,len(rows));for _,row:=range rows{out=append(out,fromList(row))};return out,total,nil}
func (s *Store) Get(ctx context.Context,id int64)(paymentreceipt.Receipt,error){row,err:=s.q.GetPaymentReceipt(ctx,id);if errors.Is(err,pgx.ErrNoRows){return paymentreceipt.Receipt{},paymentreceipt.ErrNotFound};if err!=nil{return paymentreceipt.Receipt{},err};return fromGet(row),nil}
func numeric(v *string)pgtype.Numeric{if v==nil||*v==""{return pgtype.Numeric{}};var n pgtype.Numeric;if err:=n.Scan(*v);err!=nil{return pgtype.Numeric{}};return n}
func (s *Store) Create(ctx context.Context,in paymentreceipt.CreateInput)(paymentreceipt.Receipt,error){created,err:=s.q.CreatePaymentReceipt(ctx,dbgen.CreatePaymentReceiptParams{RequestID:in.RequestID,DepartmentID:in.DepartmentID,SourceID:optID(in.SourceID),CategoryID:optID(in.CategoryID),UnitID:optID(in.UnitID),TanggalPembelian:pgtype.Date{Time:in.TanggalPembelian,Valid:true},Usage:in.Usage,DeskripsiItems:in.DeskripsiItems,JumlahItems:in.JumlahItems,Hargaitems:in.HargaItems,PriceEstimate:numeric(in.PriceEstimate),Keterangan:in.Keterangan});if err!=nil{return paymentreceipt.Receipt{},err};return s.Get(ctx,created.ID)}
